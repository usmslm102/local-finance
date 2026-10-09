package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"local-finance/internal/api"
)

func TestSampleDownloadsWithoutLocalFiles(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/samples", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("sample list status = %d", list.Code)
	}
	var samples []api.SampleStatementInfo
	if err := json.Unmarshal(list.Body.Bytes(), &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples) == 0 {
		t.Fatal("sample list is empty")
	}
	expected := make(map[string][]byte, len(samples))
	for _, sample := range samples {
		data, err := os.ReadFile("../../" + sample.Path)
		if err != nil {
			t.Fatal(err)
		}
		expected[sample.Path] = data
	}

	// Model an installed release launched outside the source checkout.
	t.Chdir(t.TempDir())
	for _, sample := range samples {
		t.Run(sample.Filename, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/samples/download?path="+url.QueryEscape(sample.Path), nil))
			if response.Code != http.StatusOK {
				t.Fatalf("download status = %d, body = %s", response.Code, response.Body.String())
			}
			if !bytes.Equal(response.Body.Bytes(), expected[sample.Path]) {
				t.Fatal("download does not match sample fixture")
			}
		})
	}

	for _, tc := range []struct {
		path string
		code int
	}{
		{"", http.StatusBadRequest},
		{"samples/credit_cards/missing.pdf", http.StatusNotFound},
		{"samples/../../go.mod", http.StatusForbidden},
		{"/etc/passwd", http.StatusForbidden},
		{"frontend/dist/index.html", http.StatusForbidden},
		{"samples/.DS_Store", http.StatusNotFound},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/samples/download?path="+url.QueryEscape(tc.path), nil))
			if response.Code != tc.code {
				t.Fatalf("status = %d, want %d", response.Code, tc.code)
			}
		})
	}
}
