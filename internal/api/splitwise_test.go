package api_test

import (
	"bytes"
	"encoding/json"
	"local-finance/internal/models"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSplitwiseUploadPreviewConfirmAndProtection(t *testing.T) {
	_, router, d, cleanup := setupTestRouter(t)
	defer cleanup()
	data, err := os.ReadFile("../../samples/splitwise/fictional-group.csv")
	if err != nil {
		t.Fatal(err)
	}
	upload := func(path string, payload []byte) *httptest.ResponseRecorder {
		t.Helper()
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		file, err := writer.CreateFormFile("file", "fictional.csv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(payload); err != nil {
			t.Fatal(err)
		}
		writer.WriteField("person", "Sanjay")
		writer.WriteField("group", "Example")
		writer.Close()
		req := localTestRequest("POST", path, body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := upload("/api/splitwise/preview", data); w.Code != 200 {
		t.Fatalf("preview: %d %s", w.Code, w.Body)
	}
	entries, _ := d.ListSplitwise()
	if len(entries) != 0 {
		t.Fatal("preview wrote data")
	}
	if w := upload("/api/splitwise/import", data); w.Code != 200 {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
	if w := upload("/api/splitwise/import", data); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"duplicates":5`)) {
		t.Fatalf("duplicate: %d %s", w.Code, w.Body)
	}
	entries, _ = d.ListSplitwise()
	var other models.SplitwiseEntry
	for _, e := range entries {
		if e.Net == -5000 {
			other = e
		}
	}
	confirm := localTestRequest("POST", "/api/splitwise/"+other.ID+"/confirm", bytes.NewBufferString(`{"share_cents":5000,"category_id":"cat_travel"}`))
	confirm.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, confirm)
	if w.Code != 204 {
		t.Fatalf("confirm: %d %s", w.Code, w.Body)
	}
	result, err := d.GetAnalyticsOverview()
	if err != nil || result.TotalExpense != 50 || len(result.CategoryBreakdown) != 1 || result.CategoryBreakdown[0].CategoryID != "cat_travel" {
		t.Fatalf("expense projection: %+v %v", result, err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest("GET", "/api/splitwise", nil))
	var list []models.SplitwiseEntry
	if err := json.Unmarshal(w.Body.Bytes(), &list); w.Code != 200 || err != nil || len(list) != 5 {
		t.Fatalf("list: %d %v", w.Code, err)
	}
	if w := upload("/api/splitwise/import", []byte("broken")); w.Code != 400 {
		t.Fatalf("invalid CSV: %d", w.Code)
	}
	if w := upload("/api/splitwise/import", bytes.Repeat([]byte("x"), (10<<20)+1)); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize upload: %d", w.Code)
	}
	w = httptest.NewRecorder()
	req := localTestRequest("GET", "/api/splitwise", nil)
	req.Header.Set("Origin", "https://attacker.example")
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("foreign origin accepted: %d", w.Code)
	}
	setup := localTestRequest("POST", "/api/auth/setup", bytes.NewBufferString(`{"password":"example-password"}`))
	setup.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, setup)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest("GET", "/api/splitwise", nil))
	if w.Code != 401 {
		t.Fatalf("Splitwise data accessible while locked: %d", w.Code)
	}
}
