package savings

import (
	"testing"
	"time"

	"bank-service/internal/modules/account"
)

func TestGetProductsReturnsSupportedTerms(t *testing.T) {
	products := (&Service{}).GetProducts()
	if len(products) != 3 {
		t.Fatalf("expected 3 savings products, got %d", len(products))
	}
	for index, term := range []int{3, 6, 12} {
		if products[index].TermMonths != term {
			t.Fatalf("expected term %d, got %d", term, products[index].TermMonths)
		}
		if products[index].InterestRate <= 0 {
			t.Fatalf("interest rate for %d months must be positive", term)
		}
	}
}

func TestMapSavingsResponseCalculatesMaturityAmount(t *testing.T) {
	detail := SavingsDetail{
		Account: account.Account{
			AccountNumber: "970412345678",
			Status:        "ACTIVE",
		},
		OriginalPrincipal:   10_000_000,
		InterestRate:        6,
		TermMonths:          6,
		StartDate:           time.Now(),
		EndDate:             time.Now().AddDate(0, 6, 0),
		MaturityInstruction: "PAYOUT",
	}

	response := mapSavingsResponse(detail)
	if response.ExpectedInterest != 300_000 {
		t.Fatalf("expected interest 300000, got %d", response.ExpectedInterest)
	}
	if response.MaturityAmount != 10_300_000 {
		t.Fatalf("expected maturity amount 10300000, got %d", response.MaturityAmount)
	}
}
