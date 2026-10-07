package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"local-finance/internal/models"
)

// SessionInfo tracks active authentication sessions
type SessionInfo struct {
	ExpiresAt time.Time
}

// SessionManager manages in-memory authentication sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]SessionInfo
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]SessionInfo),
	}

	// Periodic cleanup of expired sessions
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			sm.CleanupExpired()
		}
	}()

	return sm
}

func (sm *SessionManager) CreateSession(autoLockMinutes int) (string, time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	duration := time.Duration(autoLockMinutes) * time.Minute
	if duration <= 0 {
		duration = 60 * time.Minute
	}
	expiresAt := time.Now().Add(duration)

	sm.sessions[token] = SessionInfo{
		ExpiresAt: expiresAt,
	}

	return token, expiresAt
}

func (sm *SessionManager) ValidateSession(token string) bool {
	if token == "" {
		return false
	}

	sm.mu.RLock()
	info, exists := sm.sessions[token]
	sm.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(info.ExpiresAt) {
		sm.RevokeSession(token)
		return false
	}

	return true
}

func (sm *SessionManager) RevokeSession(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

func (sm *SessionManager) CleanupExpired() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for token, info := range sm.sessions {
		if now.After(info.ExpiresAt) {
			delete(sm.sessions, token)
		}
	}
}

// Password helpers
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func extractToken(c *gin.Context) string {
	// 1. Check Authorization header: Bearer <token>
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 2. Check HttpOnly cookie
	if cookie, err := c.Cookie("local_finance_session"); err == nil && cookie != "" {
		return cookie
	}

	return ""
}

// AuthMiddleware protects all /api endpoints when local auth is enabled
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		settings, _, err := h.db.GetSecuritySettings()
		if err != nil && strings.HasPrefix(c.Request.URL.Path, "/api/mcp/") {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to read security settings"})
			return
		}
		if err != nil || !settings.AuthEnabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// Whitelist public endpoints
		if path == "/api/health" ||
			path == "/api/system/version" ||
			path == "/api/auth/status" ||
			path == "/api/auth/login" ||
			path == "/api/auth/setup" {
			c.Next()
			return
		}

		token := extractToken(c)
		if token == "" || !h.sessionManager.ValidateSession(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "AUTH_REQUIRED",
			})
			return
		}

		c.Next()
	}
}

// GetAuthStatus returns the current local auth configuration and session state
func (h *Handler) GetAuthStatus(c *gin.Context) {
	settings, _, err := h.db.GetSecuritySettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read security settings: " + err.Error()})
		return
	}

	isAuthenticated := false
	if settings.AuthEnabled {
		token := extractToken(c)
		if token != "" && h.sessionManager.ValidateSession(token) {
			isAuthenticated = true
		}
	} else {
		isAuthenticated = true
	}

	c.JSON(http.StatusOK, models.AuthStatusResponse{
		AuthEnabled:     settings.AuthEnabled,
		IsAuthenticated: isAuthenticated,
		AutoLockMinutes: settings.AutoLockMinutes,
		FilePermissions: settings.FilePermissions,
	})
}

// SetupAuth configures the master password for the first time and enables local auth
func (h *Handler) SetupAuth(c *gin.Context) {
	var req models.AuthSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	trimmed := strings.TrimSpace(req.Password)
	if len(trimmed) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password / PIN must be at least 4 characters"})
		return
	}

	hash, err := HashPassword(trimmed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	autoLock := req.AutoLockMinutes
	if autoLock <= 0 {
		autoLock = 60
	}

	if err := h.db.SetSecurityPassword(hash, autoLock); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store security settings: " + err.Error()})
		return
	}

	// Create session for the user immediately
	token, expiresAt := h.sessionManager.CreateSession(autoLock)
	c.SetCookie("local_finance_session", token, int(autoLock*60), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Local authentication enabled successfully",
		"token":      token,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// Login verifies the master password and creates a new session
func (h *Handler) Login(c *gin.Context) {
	var req models.AuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	settings, hash, err := h.db.GetSecuritySettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check security settings"})
		return
	}

	if !settings.AuthEnabled || hash == "" {
		c.JSON(http.StatusOK, gin.H{"message": "Local auth is not enabled"})
		return
	}

	if !VerifyPassword(req.Password, hash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect master password or PIN"})
		return
	}

	autoLock := settings.AutoLockMinutes
	if autoLock <= 0 {
		autoLock = 60
	}

	token, expiresAt := h.sessionManager.CreateSession(autoLock)
	c.SetCookie("local_finance_session", token, int(autoLock*60), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Unlocked successfully",
		"token":      token,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// Logout terminates the current session and clears the session cookie
func (h *Handler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token != "" {
		h.sessionManager.RevokeSession(token)
	}

	// Clear session cookie
	c.SetCookie("local_finance_session", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ChangePassword updates the master password after verifying the existing one
func (h *Handler) ChangePassword(c *gin.Context) {
	var req models.AuthChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current and new password are required"})
		return
	}

	settings, hash, err := h.db.GetSecuritySettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check security settings"})
		return
	}

	if !settings.AuthEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Local auth is not enabled"})
		return
	}

	if !VerifyPassword(req.CurrentPassword, hash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	trimmedNew := strings.TrimSpace(req.NewPassword)
	if len(trimmedNew) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password / PIN must be at least 4 characters"})
		return
	}

	newHash, err := HashPassword(trimmedNew)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	if err := h.db.SetSecurityPassword(newHash, settings.AutoLockMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Master password updated successfully"})
}

// DisableAuth turns off local authentication after verifying the master password
func (h *Handler) DisableAuth(c *gin.Context) {
	var req models.AuthDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password confirmation is required to disable security"})
		return
	}

	_, hash, err := h.db.GetSecuritySettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check security settings"})
		return
	}

	if !VerifyPassword(req.Password, hash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password. Security was not disabled."})
		return
	}

	if err := h.db.DisableSecurity(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable security: " + err.Error()})
		return
	}

	// Revoke current session & cookie
	token := extractToken(c)
	if token != "" {
		h.sessionManager.RevokeSession(token)
	}
	c.SetCookie("local_finance_session", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Local authentication disabled successfully"})
}

// UpdateSecuritySettings updates timeout preferences
func (h *Handler) UpdateSecuritySettings(c *gin.Context) {
	var req models.UpdateSecuritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid auto lock duration"})
		return
	}

	if err := h.db.UpdateSecuritySettings(req.AutoLockMinutes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update security settings: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Security settings updated successfully"})
}
