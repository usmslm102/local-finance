package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	localfinance "local-finance"
	"local-finance/internal/app"
	"local-finance/internal/updater"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	defaultPath, err := app.DefaultDatabasePath()
	if err != nil {
		return err
	}
	dbPath := flag.String("db", defaultPath, "Path to SQLite database file")
	flag.Parse()
	path, err := app.CanonicalDatabasePath(*dbPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	var window *application.WebviewWindow
	var activeWindow atomic.Pointer[application.WebviewWindow]
	// Acquire the desktop instance lock before opening/migrating SQLite.
	// This coordinates desktop launches, not a separately launched server.
	native := application.New(application.Options{
		Name:        "LocalFinance",
		Description: "Offline personal finance",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: fmt.Sprintf("localfinance-desktop-%x", sha256.Sum256([]byte(path))),
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if current := activeWindow.Load(); current != nil {
					current.Show()
					current.Focus()
				}
			},
		},
	})

	core, err := app.OpenDesktop(path, localfinance.GetStaticFS(), &desktopActions{native: native, databasePath: path})
	if err != nil {
		return showStartupError(native, err)
	}
	listener, err := listenDesktop(path)
	if err != nil {
		core.Close()
		return showStartupError(native, err)
	}
	url := "http://" + listener.Addr().String()
	server := &http.Server{
		Handler:           core.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       time.Minute,
	}
	var closeOnce sync.Once
	shutdown := func() {
		closeOnce.Do(func() {
			core.StopMCP()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("UI shutdown: %v", err)
				_ = server.Close()
			}
			if err := core.Close(); err != nil {
				log.Printf("Database close: %v", err)
			}
		})
	}
	defer shutdown()
	// On macOS the framework can terminate without Run returning.
	native.OnShutdown(shutdown)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("UI server: %v", err)
			native.Quit()
		}
	}()
	log.Printf("LocalFinance %s; database: %s", updater.CurrentVersion, path)
	log.Printf("Shared Gin API/browser preview: %s", url)
	log.Printf("Finance MCP is disabled by default; enable it in Settings → AI / MCP. Closing the window stops it.")

	window = native.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "localfinance",
		Title:              "LocalFinance",
		URL:                url,
		Width:              1440,
		Height:             940,
		MinWidth:           900,
		MinHeight:          640,
		UseApplicationMenu: true,
		BackgroundColour:   application.NewRGB(248, 249, 251),
	})
	activeWindow.Store(window)
	refresh := func() {
		window.ExecJS("window.dispatchEvent(new Event('localfinance:refresh'))")
	}
	window.OnWindowEvent(events.Common.WindowFocus, func(*application.WindowEvent) { refresh() })
	menu := native.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}
	menu.AddRole(application.EditMenu)
	view := menu.AddSubmenu("View")
	view.Add("Refresh finance data").SetAccelerator("CmdOrCtrl+r").OnClick(func(*application.Context) { refresh() })
	view.Add("Reload window").SetAccelerator("CmdOrCtrl+Shift+r").OnClick(func(*application.Context) { window.Reload() })
	menu.AddRole(application.WindowMenu)
	if runtime.GOOS != "darwin" {
		menu.AddSubmenu("File").Add("Quit").SetAccelerator("CmdOrCtrl+q").OnClick(func(*application.Context) { native.Quit() })
	}
	native.Menu.Set(menu)
	return native.Run()
}

// Finder/GUI launches have no terminal: make ownership and startup failures visible.
func showStartupError(native *application.App, err error) error {
	native.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			native.Dialog.Error().SetTitle("LocalFinance could not start").SetMessage(err.Error()).Show()
			native.Quit()
		}()
	})
	if runErr := native.Run(); runErr != nil {
		return runErr
	}
	return err
}
