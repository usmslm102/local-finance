package parser

import (
	"testing"

	"local-finance/internal/models"
)

func TestCleanNarrationSelfTransfers(t *testing.T) {
	for _, tc := range []struct {
		name, narration string
		mode            models.PaymentMode
		transfer        bool
	}{
		{"plain", "SELF TRANSFER", models.PaymentModeOther, true},
		{"upi debit", "UPI/DR/123456789012/ALEX/HDFC/alex@okhdfcbank/SELF TRANSFER", models.PaymentModeUPI, true},
		{"upi credit", "UPI/CR/123456789012/ALEX/HDFC/alex@okhdfcbank/self transfer", models.PaymentModeUPI, true},
		{"upi hyphen", "UPI-ALEX-alex@okhdfcbank-HDFC-123456789012-SELF TRANSFER", models.PaymentModeUPI, true},
		{"imps", "IMPS-123456789012-ALEX-SELF TRANSFER", models.PaymentModeIMPS, true},
		{"neft", "NEFT CR-ABC123-ALEX-SELF TRANSFER", models.PaymentModeNEFT, true},
		{"rtgs", "RTGS DR-ABC123-ALEX-SELF TRANSFER", models.PaymentModeRTGS, true},
		{"upi purchase", "UPI/DR/123456789012/SHOP/HDFC/shop@okhdfcbank/PAYMENT", models.PaymentModeUPI, false},
		{"imps payment", "IMPS-123456789012-ALEX-PAYMENT", models.PaymentModeIMPS, false},
		{"neft salary", "NEFT CR-ABC123-EMPLOYER-SALARY", models.PaymentModeSalary, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CleanNarration(tc.narration)
			if got.IsTransfer != tc.transfer {
				t.Errorf("IsTransfer = %v; want %v", got.IsTransfer, tc.transfer)
			}
			if got.PaymentMode != tc.mode {
				t.Errorf("PaymentMode = %s; want %s", got.PaymentMode, tc.mode)
			}
		})
	}
	got := CleanNarration("UPI/DR/123456789012/ALEX/HDFC/alex@okhdfcbank/SELF TRANSFER")
	if got.ReferenceNumber != "123456789012" || got.UPIVPA == nil || *got.UPIVPA != "alex@okhdfcbank" || got.CleanedPayee != "Alex" {
		t.Fatalf("self-transfer payment metadata lost: %+v", got)
	}
}
