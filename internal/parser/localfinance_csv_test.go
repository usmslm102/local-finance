package parser

import (
	"strings"
	"testing"
)

const fictionalCSVHeader = "bank_name,account_type,account_number_mask,date,narration,amount,tx_type,reference_number,running_balance\n"
const fictionalCSVRow = "Fictional Bank,SAVINGS,XXXX1234,2026-01-10,Corner Shop,125.50,DEBIT,ref-1,-25.50\n"

func TestLocalFinanceCSVStrictRowsAndDetection(t *testing.T) {
	p := &LocalFinanceCSVParser{}
	input := fictionalCSVHeader + fictionalCSVRow
	txs, meta, err := p.Parse(strings.NewReader("\ufeff"+input), ParseOptions{})
	if err != nil || len(txs) != 1 || txs[0].Amount != 125.5 || *txs[0].RunningBalance != -25.5 || txs[0].ReferenceNumber != "ref-1" || meta.TotalDebits != 125.5 || meta.StartDate != "2026-01-10" {
		t.Fatalf("parse: %+v %+v %v", txs, meta, err)
	}
	detected, confidence, _ := DefaultRegistry.Detect("transcription.csv", []byte(input))
	if detected == nil || detected.ID() != p.ID() || confidence != 1 {
		t.Fatalf("detection: %v %v", detected, confidence)
	}
	quoted := fictionalCSVHeader + "Fictional Bank,SAVINGS,XXXX1234,2026-01-10,\"Corner, Shop\nReceipt\",125.50,DEBIT,ref-1,\n"
	if txs, _, err := p.Parse(strings.NewReader(quoted), ParseOptions{}); err != nil || txs[0].RawNarration != "Corner, Shop\nReceipt" {
		t.Fatalf("quoted narration: %+v %v", txs, err)
	}
	withReference := strings.Replace(input, "Corner Shop", "UPI/DR/123456789012/Shop/Bank/shop@ybl", 1)
	txs, _, err = p.Parse(strings.NewReader(withReference), ParseOptions{})
	if err != nil || txs[0].ReferenceNumber != "ref-1" {
		t.Fatalf("explicit reference lost: %+v %v", txs, err)
	}
	txs, _, err = p.Parse(strings.NewReader(strings.Replace(withReference, ",ref-1,", ",,", 1)), ParseOptions{})
	if err != nil || txs[0].ReferenceNumber != "123456789012" {
		t.Fatalf("reference fallback lost: %+v %v", txs, err)
	}
	for name, invalid := range map[string]string{
		"empty":     fictionalCSVHeader,
		"missing":   strings.Replace(input, "bank_name,", "", 1),
		"unknown":   strings.Replace(input, "bank_name,", "unknown,", 1),
		"duplicate": strings.Replace(input, "bank_name,", "date,", 1),
		"width":     input + "Fictional Bank,SAVINGS\n",
		"mixed":     input + strings.Replace(fictionalCSVRow, "XXXX1234", "XXXX9999", 1),
		"date":      strings.Replace(input, "2026-01-10", "2026-02-30", 1),
		"type":      strings.Replace(input, "DEBIT", "DR", 1),
		"kind":      strings.Replace(input, "SAVINGS", "OTHER", 1),
		"blank":     strings.Replace(input, "Corner Shop", " ", 1),
		"negative":  strings.Replace(input, "125.50", "-125.50", 1),
		"zero":      strings.Replace(input, "125.50", "0", 1),
		"precision": strings.Replace(input, "125.50", "125.501", 1),
		"nan":       strings.Replace(input, "125.50", "NaN", 1),
		"exponent":  strings.Replace(input, "125.50", "1e2", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := p.Parse(strings.NewReader(invalid), ParseOptions{}); err == nil {
				t.Fatal("accepted invalid CSV")
			}
		})
	}
}
