package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"local-finance/internal/app"
)

func TestUpdaterShutdownTimeoutReleasesLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	core, err := app.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	entered := make(chan struct{})
	finished := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(finished)
	}))
	defer server.Close()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		response, err := server.Client().Get(server.URL)
		if err == nil {
			response.Body.Close()
		}
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // An active request forces Shutdown down the timeout/error path.
	if err := shutdownForRestart(ctx, server.Config, core); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("active request was not canceled")
	}
	<-clientDone
	replacement, err := app.Open(path, nil)
	if err != nil {
		t.Fatalf("replacement cannot acquire ledger: %v", err)
	}
	replacement.Close()
}
