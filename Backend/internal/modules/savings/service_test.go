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

func TestCalculateTermInterestRoundsToWholeVND(t *testing.T) {
	interest := calculateTermInterest(5_000_001, 6.2, 3)
	if interest != 77_500 {
		t.Fatalf("expected rounded interest 77500, got %d", interest)
	}
}

func TestMapSavingsResponseIncludesMaturityHistory(t *testing.T) {
	processedAt := time.Now()
	detail := SavingsDetail{
		Account: account.Account{
			AccountNumber: "970412345678",
			Status:        "ACTIVE",
		},
		OriginalPrincipal: 10_000_000,
		InterestRate:      7,
		TermMonths:        6,
		RenewalCount:      1,
	}
	event := SavingsMaturityEvent{
		CycleNumber:  1,
		Principal:    10_000_000,
		Interest:     350_000,
		InterestRate: 7,
		TermMonths:   6,
		Outcome:      "RENEWED",
		ProcessedAt:  processedAt,
	}

	response := mapSavingsResponse(detail, []SavingsMaturityEvent{event})
	if response.RenewalCount != 1 {
		t.Fatalf("expected renewal count 1, got %d", response.RenewalCount)
	}
	if len(response.MaturityHistory) != 1 {
		t.Fatalf(
			"expected one maturity event, got %d",
			len(response.MaturityHistory),
		)
	}
	if response.MaturityHistory[0].Outcome != "RENEWED" {
		t.Fatalf(
			"expected RENEWED outcome, got %s",
			response.MaturityHistory[0].Outcome,
		)
	}
}

func TestCalculateDemandInterestUsesActualDays(t *testing.T) {
	interest := calculateDemandInterest(10_000_000, 0.5, 30)
	if interest != 4_110 {
		t.Fatalf("expected demand interest 4110, got %d", interest)
	}
}

func TestDerivedInterestIdempotencyKeyIsStableAndBounded(t *testing.T) {
	first := derivedInterestIdempotencyKey(
		"withdrawal-request-with-a-valid-long-idempotency-key",
	)
	second := derivedInterestIdempotencyKey(
		"withdrawal-request-with-a-valid-long-idempotency-key",
	)
	if first != second {
		t.Fatal("derived idempotency key must be deterministic")
	}
	if len(first) > 64 {
		t.Fatalf("derived key exceeds database limit: %d", len(first))
	}
}
