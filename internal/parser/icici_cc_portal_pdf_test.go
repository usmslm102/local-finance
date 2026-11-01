package parser

import (
	"bytes"
	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
	"local-finance/internal/testutil"
	"testing"
)

func TestICICICCPortalDetectionAndParsing(t *testing.T) {
	data := testutil.ICICIPortalPDF(t)
	for _, name := range []string{"CCLastStatement29-10-2026.pdf", "statement.pdf"} {
		p, confidence, _ := DefaultRegistry.Detect(name, data)
		if p == nil || p.ID() != "icici_cc_pdf_v1" || confidence < 0.9 {
			t.Fatalf("portal detection: %v %v", p, confidence)
		}
		txs, meta, err := p.Parse(bytes.NewReader(data), ParseOptions{Filename: name})
		if err != nil {
			t.Fatal(err)
		}
		if len(txs) != 3 || txs[0].Amount != 100 || txs[0].TxType != models.TxTypeDebit || txs[0].RawNarration != "EXAMPLE MERCHANT SHOP STORE MUMBAI" || txs[0].ReferenceNumber != "12345678901" || txs[0].RewardPointsEarned != 2 || txs[0].CashbackAmount != 0 || txs[1].IsTransfer || !txs[2].IsTransfer {
			t.Fatalf("portal transactions: %+v", txs)
		}
		if meta.StartDate != "2026-10-01" || meta.EndDate != "2026-10-29" || meta.StatementDate != "2026-10-29" || meta.PaymentDueDate != "2026-11-15" || meta.TotalDueAmount != 100 || meta.TotalDebits != 100 || meta.TotalCredits != 50 || meta.AccountHolderName != "EXAMPLE CARDHOLDER" || meta.CardNetwork != "" || meta.CardVariant != "" || meta.AccountNumberMask != "" {
			t.Fatalf("portal metadata: %+v", meta)
		}
	}
}

func TestICICICCPortalRejectsIncompleteOrContradictoryRows(t *testing.T) {
	data := testutil.ICICIPortalPDF(t)
	for _, change := range []func([]extractor.PositionalRow){
		func(rows []extractor.PositionalRow) {
			for i := range rows {
				for j := range rows[i].Elements {
					if rows[i].Elements[j].S == "100 Dr." {
						rows[i].Elements[j].S = "99 Dr."
						return
					}
				}
			}
		},
		func(rows []extractor.PositionalRow) {
			for i := range rows {
				for j := range rows[i].Elements {
					if rows[i].Elements[j].S == "100 Dr." {
						rows[i].Elements[j].S = "100"
						return
					}
				}
			}
		},
		func(rows []extractor.PositionalRow) {
			for i := range rows {
				for j := range rows[i].Elements {
					if rows[i].Elements[j].S == "27-10-2026" {
						rows[i].Elements[j].S = "31-10-2026"
						return
					}
				}
			}
		},
	} {
		rows, err := extractor.ExtractPDFPositionalRows(bytes.NewReader(data), "")
		if err != nil {
			t.Fatal(err)
		}
		change(rows)
		if _, _, err := parseICICICCPortalRows(rows); err == nil {
			t.Fatal("invalid portal row accepted")
		}
	}
	if confidence, _, _ := (&ICICICCPDFParser{}).CanParse("statement.pdf", []byte("%PDF Transaction Date Details Amount (INR) Reward Points Reference Number")); confidence != 0 {
		t.Fatal("generic table assigned to ICICI")
	}
}
