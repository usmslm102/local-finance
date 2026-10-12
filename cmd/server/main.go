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
	"os/signal"
	"runtime"
	"syscall"
	"time"

	localfinance "local-finance"
	"local-finance/internal/app"
	"local-finance/internal/updater"
)

func main() {
	port := flag.Int("port", 8080, "Port to run HTTP server on")
	dbPath := flag.String("db", "", "Path to SQLite database file (default: ~/.localfinance/local_finance.db)")
	openBrowserFlag := flag.Bool("open", true, "Auto-open web app in default browser (default: true; use -open=false to disable)")
	flag.Parse()

	finalDBPath := *dbPath
	if finalDBPath == "" {
		var err error
		finalDBPath, err = app.DefaultDatabasePath()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("📁 Database location: %s", finalDBPath)
	application, err := app.Open(finalDBPath, localfinance.GetStaticFS())
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer application.Close()

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
		Addr:              actualAddr,
		Handler:           application.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	// Register server details and graceful shutdown hook with updater
	updater.RegisterServerContext(actualPort, finalDBPath, func(ctx context.Context) error {
		log.Printf("🛑 Releasing listener and shutting down HTTP server on %s for updater restart...", actualAddr)
		return shutdownForRestart(ctx, srv, application)
	})

	log.Printf("🚀 Local Finance server listening on %s", url)

	if *openBrowserFlag {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	go func() {
		<-signalCtx.Done()
		application.StopMCP()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			_ = srv.Close()
		}
	}()
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

// Force-close connections after the grace period before releasing ledger ownership.
// A replacement must never start while this process still owns the database.
func shutdownForRestart(ctx context.Context, srv *http.Server, application *app.App) error {
	application.StopMCP()
	var closeErr error
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Graceful updater shutdown failed; closing connections: %v", err)
		closeErr = srv.Close()
	}
	return errors.Join(closeErr, application.Close())
}
