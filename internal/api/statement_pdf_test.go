package api_test

import (
	"bytes"
	"encoding/json"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"local-finance/internal/models"
)

func TestImportedStatementPDF(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	original, err := os.ReadFile("../../samples/savings/HDFC_Savings_Account_Statement.pdf")
	if err != nil {
		t.Fatal(err)
	}
	var encrypted bytes.Buffer
	if err := pdfapi.Encrypt(bytes.NewReader(original), &encrypted, model.NewAESConfiguration("test-password", "test-password", 256)); err != nil {
		t.Fatal(err)
	}

	for _, sample := range []struct {
		name     string
		data     []byte
		password string
	}{
		{"statement.pdf", original, ""},
		{"protected.pdf", encrypted.Bytes(), "test-password"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			file, err := form.CreateFormFile("file", sample.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(sample.data); err != nil {
				t.Fatal(err)
			}
			if err := form.WriteField("password", sample.password); err != nil {
				t.Fatal(err)
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			req := localTestRequest("POST", "/api/statements/upload", &body)
			req.Header.Set("Content-Type", form.FormDataContentType())
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != http.StatusOK {
				t.Fatalf("upload: %d %s", response.Code, response.Body.String())
			}
			var result models.ImportResult
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if sample.password != "" && result.InsertedCount != 0 {
				t.Fatalf("re-import created %d duplicates", result.InsertedCount)
			}

			response = httptest.NewRecorder()
			router.ServeHTTP(response, localTestRequest("GET", "/api/statements/"+result.StatementImportID+"/pdf", nil))
			if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), sample.data) {
				t.Fatalf("source PDF differs: status %d", response.Code)
			}
			if response.Header().Get("Content-Type") != "application/pdf" || response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("unsafe preview headers: %v", response.Header())
			}
			disposition, params, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
			if err != nil || disposition != "inline" || params["filename"] != sample.name {
				t.Fatalf("unexpected PDF disposition: %s (%v)", response.Header().Get("Content-Disposition"), err)
			}
		})
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, localTestRequest("GET", "/api/statements", nil))
	var history []models.StatementImport
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || !history[0].HasSourcePDF || !history[1].HasSourcePDF {
		t.Fatalf("unexpected PDF history: %+v", history)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"source_pdf":`)) {
		t.Fatal("history must not serialize PDF bytes")
	}

	// Older imports and non-PDF imports have no source, and unknown IDs do not expose files.
	if err := database.CreateStatementImport(&models.StatementImport{ID: "legacy", AccountID: history[0].AccountID, Filename: "old.pdf", StatementFormat: "PDF", ImportedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"legacy", "missing"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, localTestRequest("GET", "/api/statements/"+id+"/pdf", nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", id, response.Code)
		}
	}

	// A saved document must require the same session as the rest of the ledger.
	response = httptest.NewRecorder()
	req := localTestRequest("POST", "/api/auth/setup", bytes.NewBufferString(`{"password":"preview-test-password","auto_lock_minutes":30}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("auth setup: %s", response.Body.String())
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &auth); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	req = localTestRequest("GET", "/api/statements/"+history[0].ID+"/pdf", nil)
	router.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	response = httptest.NewRecorder()
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized preview: %d", response.Code)
	}

	if err := database.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := database.GetStatementPDF(history[0].ID); err == nil {
		t.Fatal("reset retained statement PDF")
	}
}
