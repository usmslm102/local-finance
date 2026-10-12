// Package app composes LocalFinance's shared Go application for browser and desktop hosts.
// Hosts own their UI/HTTP lifecycle and must drain UI requests before closing App.
package app

import (
	"io/fs"
	"log"
	"net/http"
	"sync"

	"local-finance/internal/api"
	"local-finance/internal/db"
	"local-finance/internal/mcp"
	"local-finance/internal/service"
)

// App owns one database and its optional finance MCP listener. Handler preserves
// the existing Gin routes, authentication and import semantics for either host.
type App struct {
	Handler   http.Handler
	database  *db.DB
	mcp       *mcp.Manager
	lock      *databaseLock
	closeOnce sync.Once
	closeErr  error
}

func Open(dbPath string, assets fs.FS) (*App, error) {
	return open(dbPath, assets, nil)
}

func OpenDesktop(dbPath string, assets fs.FS, desktop api.DesktopActions) (*App, error) {
	return open(dbPath, assets, desktop)
}

func open(dbPath string, assets fs.FS, desktop api.DesktopActions) (*App, error) {
	path, err := CanonicalDatabasePath(dbPath)
	if err != nil {
		return nil, err
	}
	lock, err := acquireDatabaseLock(path)
	if err != nil {
		return nil, err
	}
	database, err := db.NewDB(path)
	if err != nil {
		lock.Close()
		return nil, err
	}
	manager := mcp.NewManager(database)
	if err := manager.Start(); err != nil {
		// A configured MCP port failure must not prevent using the UI.
		log.Printf("MCP unavailable: %v", err)
	}
	var handler http.Handler
	if desktop != nil {
		handler = api.SetupDesktopRouter(database, service.NewTransactionService(database), assets, desktop, manager)
	} else {
		handler = api.SetupRouter(database, service.NewTransactionService(database), assets, manager)
	}
	return &App{
		Handler:  handler,
		lock:     lock,
		database: database,
		mcp:      manager,
	}, nil
}

// StopMCP cancels and drains requests before updater shutdown or data replacement.
func (a *App) StopMCP() { a.mcp.Close() }

// Close must be called after the host has stopped and drained its UI requests.
func (a *App) Close() error {
	a.closeOnce.Do(func() {
		a.StopMCP()
		a.closeErr = a.database.Close()
		a.lock.Close()
	})
	return a.closeErr
}
