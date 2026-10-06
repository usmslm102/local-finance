package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	localfinance "local-finance"
	"local-finance/internal/api"
	"local-finance/internal/db"
	"local-finance/internal/service"
	"local-finance/internal/updater"
)

func main() {
	port := flag.Int("port", 8080, "Port to run HTTP server on")
	dbPath := flag.String("db", "", "Path to SQLite database file (default: ~/.localfinance/local_finance.db)")
	openBrowserFlag := flag.Bool("open", true, "Auto-open web app in default browser (default: true; use -open=false to disable)")
	flag.Parse()

	// Default database path in user home directory if not provided
	finalDBPath := *dbPath
	if finalDBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			finalDBPath = "./local_finance.db"
		} else {
			appDir := filepath.Join(homeDir, ".localfinance")
			_ = os.MkdirAll(appDir, 0755)
			finalDBPath = filepath.Join(appDir, "local_finance.db")
		}
	}

	log.Printf("📁 Database location: %s", finalDBPath)
	database, err := db.NewDB(finalDBPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer database.Close()

	svc := service.NewTransactionService(database)
	router := api.SetupRouter(database, svc, localfinance.GetStaticFS())

	addr := fmt.Sprintf("127.0.0.1:%d", *port)

	// Bind TCP listener directly
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Fallback to random available port if specified port is occupied
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("❌ Failed to start listener: %v", err)
		}
	}

	actualAddr := listener.Addr().String()
	actualPort := *port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	url := fmt.Sprintf("http://%s", actualAddr)

	srv := &http.Server{
		Addr:    actualAddr,
		Handler: router,
	}

	// Register server details and graceful shutdown hook with updater
	updater.RegisterServerContext(actualPort, finalDBPath, func(ctx context.Context) error {
		log.Printf("🛑 Releasing listener and shutting down HTTP server on %s for updater restart...", actualAddr)
		return srv.Shutdown(ctx)
	})

	log.Printf("🚀 Local Finance server listening on %s", url)

	if *openBrowserFlag {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("❌ Server error: %v", err)
	}

	// Keep main alive if updater is relaunching the process
	if updater.IsRestarting() {
		log.Println("⏳ Server listener closed; keeping process alive until replacement is launched...")
		updater.WaitForRelaunch()
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if err != nil {
		log.Printf("⚠️ Could not open default browser automatically: %v", err)
	}
}
