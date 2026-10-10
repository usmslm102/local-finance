package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"local-finance/internal/models"
)

func TestMonthlyReviewAPIValidationAndAuthentication(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	for _, test := range []struct {
		url    string
		status int
	}{
		{"/api/analytics/monthly-review", 200},
		{"/api/analytics/monthly-review?month=2024-02", 200},
		{"/api/analytics/monthly-review?month=2024-2", 400},
		{"/api/analytics/monthly-review?month=9999-01", 400},
		{"/api/analytics/monthly-review?month=ALL", 400},
		{"/api/analytics/monthly-review/transactions?month=2024-02&category=", 200},
		{"/api/analytics/monthly-review/transactions?month=2024-02", 400},
		{"/api/analytics/monthly-review/transactions?month=bad&category=", 400},
		{"/api/analytics/monthly-review/transactions?month=2024-02&category=&page=0", 400},
		{"/api/analytics/monthly-review/transactions?month=2024-02&category=&page=abc", 400},
		{"/api/analytics/monthly-review/transactions?month=2024-02&category=&period=all", 400},
	} {
		t.Run(test.url, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, localTestRequest(http.MethodGet, test.url, nil))
			if w.Code != test.status {
				t.Fatalf("status %d, expected %d: %s", w.Code, test.status, w.Body.String())
			}
			if test.url == "/api/analytics/monthly-review" {
				var review models.MonthlyReview
				if err := json.Unmarshal(w.Body.Bytes(), &review); err != nil {
					t.Fatal(err)
				}
				if review.Categories == nil || review.Coverage == nil || review.CoverageComplete {
					t.Fatal("empty history must be usable, without a completeness claim")
				}
			}
		})
	}
	payload, _ := json.Marshal(models.AuthSetupRequest{Password: "disposable-review-test", AutoLockMinutes: 30})
	req := localTestRequest(http.MethodPost, "/api/auth/setup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("auth setup: %s", w.Body.String())
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/api/analytics/monthly-review", "/api/analytics/monthly-review/transactions?month=2024-02&category="} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, localTestRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("review exposed without authentication: %d", w.Code)
		}
		req := localTestRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+session.Token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("authenticated review: %d", w.Code)
		}
	}
}
