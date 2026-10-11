package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"local-finance/internal/models"
)

func TestPlanningAPIValidationAndAuthentication(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	request := func(method, path, body string, token string) *httptest.ResponseRecorder {
		req := localTestRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	valid := `{"name":"Laptop","target_amount":90000,"saved_amount":10000,"monthly_contribution":5000,"target_date":"2027-10-11"}`
	if res := request("POST", "/api/goals", valid, ""); res.Code != 204 {
		t.Fatalf("create: %d %s", res.Code, res.Body)
	}
	res := request("GET", "/api/goals", "", "")
	var goals []models.SavingsGoal
	if err := json.Unmarshal(res.Body.Bytes(), &goals); err != nil || len(goals) != 1 {
		t.Fatalf("read: %s", res.Body)
	}
	if res := request("POST", "/api/goals", `{"name":"bad","target_amount":-1}`, ""); res.Code != 400 {
		t.Fatal("invalid goal accepted")
	}
	if res := request("PUT", "/api/goals/missing", valid, ""); res.Code != 404 {
		t.Fatal("update created missing goal")
	}
	if res := request("PUT", "/api/goals/"+goals[0].ID, valid, ""); res.Code != 204 {
		t.Fatalf("update: %s", res.Body)
	}
	if res := request("GET", "/api/analytics/data-health", "", ""); res.Code != 200 || res.Body.String() != "[]" {
		t.Fatalf("empty health: %s", res.Body)
	}
	if res := request("GET", "/api/transactions?review_needed=true", "", ""); res.Code != 200 {
		t.Fatal("review filter failed")
	}
	res = request("POST", "/api/auth/setup", `{"password":"planning-test-password","auto_lock_minutes":30}`, "")
	if res.Code != 200 {
		t.Fatalf("auth setup: %s", res.Body)
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/goals", "/api/analytics/data-health"} {
		if res := request("GET", path, "", ""); res.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated read exposed %s", path)
		}
		if res := request("GET", path, "", session.Token); res.Code != 200 {
			t.Fatalf("authenticated read failed %s", path)
		}
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		path := "/api/goals"
		if method != "POST" {
			path += "/" + goals[0].ID
		}
		if res := request(method, path, valid, ""); res.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated %s accepted", method)
		}
	}
	if res := request("DELETE", "/api/goals/"+goals[0].ID, "", session.Token); res.Code != 204 {
		t.Fatalf("delete: %s", res.Body)
	}
}
