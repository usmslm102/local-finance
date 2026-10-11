package api_test

import (
	"bytes"
	"encoding/json"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

func TestSplitwiseAutomaticUploadPreviewAndProtection(t *testing.T) {
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
	if w := upload("/api/splitwise/import", data); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"duplicates":4`)) {
		t.Fatalf("duplicate: %d %s", w.Code, w.Body)
	}
	wMembers := httptest.NewRecorder()
	router.ServeHTTP(wMembers, localTestRequest("GET", "/api/splitwise/members", nil))
	var members []models.SplitwiseMember
	if err := json.Unmarshal(wMembers.Body.Bytes(), &members); err != nil || wMembers.Code != 200 || len(members) != 2 {
		t.Fatalf("members: %d %s", wMembers.Code, wMembers.Body)
	}
	aliasesReq := localTestRequest("POST", "/api/splitwise/members", bytes.NewBufferString(`{"group":"Example","name":"Asha Example","aliases":["asha@fictional"]}`))
	aliasesReq.Header.Set("Content-Type", "application/json")
	wMembers = httptest.NewRecorder()
	router.ServeHTTP(wMembers, aliasesReq)
	if wMembers.Code != 204 {
		t.Fatalf("aliases: %d %s", wMembers.Code, wMembers.Body)
	}
	wMembers = httptest.NewRecorder()
	router.ServeHTTP(wMembers, localTestRequest("GET", "/api/splitwise/settlements", nil))
	if wMembers.Code != 200 || wMembers.Body.String() != "[]" {
		t.Fatalf("settlement suggestions: %d %s", wMembers.Code, wMembers.Body)
	}
	invalidSettlement := localTestRequest("POST", "/api/splitwise/settlements", bytes.NewBufferString(`{"transaction_id":"missing","group":"Example","member":"Asha Example"}`))
	invalidSettlement.Header.Set("Content-Type", "application/json")
	wMembers = httptest.NewRecorder()
	router.ServeHTTP(wMembers, invalidSettlement)
	if wMembers.Code != 400 {
		t.Fatalf("invalid settlement: %d", wMembers.Code)
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
	if err := json.Unmarshal(w.Body.Bytes(), &list); w.Code != 200 || err != nil || len(list) != 4 {
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

func TestSplitwiseReviewAPIRequiresExplicitAmountApproval(t *testing.T) {
	_, router, d, cleanup := setupTestRouter(t)
	defer cleanup()
	account, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "", "XX7744", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "review-api", TxDate: "2026-08-27", RawNarration: "Fictional retailer", Amount: 260, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Fictional purchase,General,200,INR,120,-120\n"), "API example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Review != 1 {
		t.Fatalf("import: %+v %v", result, err)
	}
	path := "/api/splitwise/" + entries[0].ID
	for _, review := range []bool{false, true} {
		suffix := ""
		if review {
			suffix = "&review=true"
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, localTestRequest("GET", path+"/candidates?share_cents="+strconv.FormatInt(entries[0].Share, 10)+suffix, nil))
		var candidates []models.Transaction
		if err := json.Unmarshal(w.Body.Bytes(), &candidates); err != nil || w.Code != 200 {
			t.Fatalf("candidates: %d %s", w.Code, w.Body)
		}
		if review && (len(candidates) != 1 || candidates[0].ID != bank.ID) || !review && len(candidates) != 0 {
			t.Fatalf("review opt-in=%v: %+v", review, candidates)
		}
	}
	confirmation := models.SplitwiseConfirmation{Share: entries[0].Share, TransactionID: bank.ID}
	for _, approve := range []bool{false, true} {
		confirmation.AllowAmountDifference = approve
		data, _ := json.Marshal(confirmation)
		req := localTestRequest("POST", path+"/confirm", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		want := 400
		if approve {
			want = 204
		}
		if w.Code != want {
			t.Fatalf("approve=%v: %d %s", approve, w.Code, w.Body)
		}
	}
	original, _ := d.GetTransaction(bank.ID)
	if original.Amount != 260 || original.PersonalExpenseAmount == nil || *original.PersonalExpenseAmount != 80 {
		t.Fatalf("approved ledger: %+v", original)
	}
}
