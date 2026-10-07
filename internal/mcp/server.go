// Package mcp exposes LocalFinance's read-only finance operations over local MCP.
package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
)

// Manager owns the independent MCP listener and its access policy.
// All lifecycle mutations are serialized; closing drains requests before DB restore.
type Manager struct {
	mu        sync.Mutex
	active    sync.WaitGroup
	database  *db.DB
	settings  db.MCPSettings
	server    *http.Server
	listener  net.Listener
	cancel    context.CancelFunc
	handler   http.Handler
	lastError string
}

type Status struct {
	Enabled   bool   `json:"enabled"`
	Port      int    `json:"port"`
	HasToken  bool   `json:"has_token"`
	Listening bool   `json:"listening"`
	Endpoint  string `json:"endpoint"`
	Error     string `json:"error,omitempty"`
}

func NewManager(database *db.DB) *Manager {
	m := &Manager{database: database}
	settings, err := database.GetMCPSettings()
	if err != nil {
		m.lastError = "Unable to read MCP settings"
	} else {
		m.settings = settings
	}
	server := newServer(database)
	m.handler = sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 1 << 20})
	return m
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	s, err := m.database.GetMCPSettings()
	if err != nil {
		m.settings.Enabled = false
		m.settings.TokenHash = ""
		m.lastError = "Unable to read MCP settings"
		return err
	}
	m.settings = s
	return m.startLocked()
}

func (m *Manager) startLocked() error {
	if !m.settings.Enabled {
		return nil
	}
	if m.settings.TokenHash == "" {
		m.lastError = "Create a token before enabling MCP"
		return fmt.Errorf("%s", m.lastError)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(m.settings.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		m.lastError = fmt.Sprintf("MCP port %d is unavailable. Choose another port and save.", m.settings.Port)
		return fmt.Errorf("%s", m.lastError)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	srv := &http.Server{Handler: m, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	m.server = srv
	m.listener = listener
	m.lastError = ""
	go func() {
		err := srv.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			m.mu.Lock()
			if m.server == srv {
				m.lastError = "MCP listener stopped unexpectedly"
				m.server = nil
				cancel()
			}
			m.mu.Unlock()
		}
	}()
	return nil
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}
func (m *Manager) statusLocked() Status {
	return Status{Enabled: m.settings.Enabled, Port: m.settings.Port, HasToken: m.settings.TokenHash != "", Listening: m.server != nil, Endpoint: fmt.Sprintf("http://127.0.0.1:%d/mcp", m.settings.Port), Error: m.lastError}
}
func (m *Manager) stopLocked() {
	// Close the raw listener too: Server.Close may run before Serve registers it.
	if m.listener != nil {
		_ = m.listener.Close()
		m.listener = nil
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.server != nil {
		_ = m.server.Close()
		m.server = nil
	}
	m.active.Wait()
}
func (m *Manager) Close() { m.mu.Lock(); defer m.mu.Unlock(); m.stopLocked() }

func (m *Manager) Configure(enabled bool, port int) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.settings
	s.Enabled, s.Port = enabled, port
	if err := m.database.SetMCPSettings(s); err != nil {
		return m.statusLocked(), err
	}
	m.stopLocked()
	m.settings = s
	m.lastError = ""
	// Listener failures are represented in Status; preferences remain saved for retry.
	_ = m.startLocked()
	return m.statusLocked(), nil
}

func (m *Manager) RotateToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("unable to create token")
	}
	token := hex.EncodeToString(secret)
	hash := sha256.Sum256([]byte(token))
	s := m.settings
	s.TokenHash = hex.EncodeToString(hash[:])
	if err := m.database.SetMCPSettings(s); err != nil {
		return "", err
	}
	m.stopLocked()
	m.settings = s
	_ = m.startLocked()
	return token, nil
}

// Revoke removes credentials and drains reads before reset/restore proceeds.
func (m *Manager) Revoke() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	m.settings.Enabled = false
	m.settings.TokenHash = ""
	return m.database.SetMCPSettings(m.settings)
}

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil || (host != "127.0.0.1" && host != "localhost") {
		http.Error(w, "Invalid MCP host", http.StatusForbidden)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || u.Host != r.Host || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
			http.Error(w, "Invalid MCP origin", http.StatusForbidden)
			return
		}
	}
	if r.URL.Path != "/mcp" {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	if !m.settings.Enabled || m.server == nil || port != strconv.Itoa(m.settings.Port) {
		m.mu.Unlock()
		http.Error(w, "MCP is disabled", http.StatusServiceUnavailable)
		return
	}
	raw := r.Header.Get("Authorization")
	token := strings.TrimPrefix(raw, "Bearer ")
	hash := sha256.Sum256([]byte(token))
	expected, err := hex.DecodeString(m.settings.TokenHash)
	if err != nil || token == raw || token == "" || subtle.ConstantTimeCompare(hash[:], expected) != 1 {
		m.mu.Unlock()
		w.Header().Set("WWW-Authenticate", `Bearer realm="LocalFinance MCP"`)
		http.Error(w, "Invalid MCP token", http.StatusUnauthorized)
		return
	}
	m.active.Add(1)
	m.mu.Unlock()
	defer m.active.Done()
	m.handler.ServeHTTP(w, r)
}
