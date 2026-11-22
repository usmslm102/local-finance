package api_test

import (
	"bytes"
	"encoding/json"
	"local-finance/internal/retirement"
	"net/http/httptest"
	"testing"
)

func TestFIREAPIParityValidationAndNoWrites(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	request := func(method, path string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		req := localTestRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	scenario := retirement.Scenario{Age: 30, Horizon: 90, MonthlyExpense: 100000, RetirementExpense: 100000, Inflation: 6, WithdrawalRate: 3, RSUVestYears: 4, RSUNetPercent: 70, Assets: []retirement.Asset{{Kind: "MARKET", Name: "Fixture", Currency: "INR", Balance: 50000000, Return: 8, AccessiblePercent: 100}}}
	before, _ := database.DataRevision()
	payload, _ := json.Marshal(scenario)
	res := request("POST", "/api/planning/fire/project", payload)
	if res.Code != 200 {
		t.Fatalf("projection: %d %s", res.Code, res.Body)
	}
	expected, _ := retirement.Project(scenario)
	var actual retirement.Result
	if err := json.Unmarshal(res.Body.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.RetirementAge == nil || *actual.RetirementAge != *expected.RetirementAge || actual.TodayTarget != 40000000 || len(actual.Points) != len(expected.Points) {
		t.Fatal("API differs from calculation")
	}
	if res = request("GET", "/api/planning/fire/baseline", nil); res.Code != 200 {
		t.Fatalf("baseline %s", res.Body)
	}
	after, _ := database.DataRevision()
	if before != after {
		t.Fatal("FIRE reads/projection modified finance data")
	}
	for _, body := range [][]byte{[]byte(`{"age":30,"unknown":true}`), append(append([]byte{}, payload...), []byte(` {}`)...), []byte(`{"age":30.5}`), []byte(`{"withdrawal_percent":0}`)} {
		if res = request("POST", "/api/planning/fire/project", body); res.Code != 400 {
			t.Fatalf("invalid scenario accepted: %s", body)
		}
	}
	scenario.WithdrawalRate = 1e-300
	payload, _ = json.Marshal(scenario)
	if res = request("POST", "/api/planning/fire/project", payload); res.Code != 400 {
		t.Fatal("numeric overflow did not return controlled error")
	}
	res = request("POST", "/api/auth/setup", []byte(`{"password":"fire-test-password","auto_lock_minutes":30}`))
	if res.Code != 200 {
		t.Fatal("setup auth failed")
	}
	for _, route := range []struct{ method, path string }{{"GET", "/api/planning/fire/baseline"}, {"POST", "/api/planning/fire/project"}} {
		if res = request(route.method, route.path, payload); res.Code != 401 {
			t.Fatalf("unauthenticated finance route %s: %d", route.path, res.Code)
		}
	}
}
