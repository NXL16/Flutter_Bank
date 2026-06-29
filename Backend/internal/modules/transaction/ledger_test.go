package transaction

import "testing"

func TestCreateDoubleEntryRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		transaction uint
		debit       uint
		credit      uint
		amount      int64
		currency    string
	}{
		{"nil transaction", 1, 1, 2, 1000, "VND"},
		{"missing transaction id", 0, 1, 2, 1000, "VND"},
		{"same account", 1, 2, 2, 1000, "VND"},
		{"non-positive amount", 1, 1, 2, 0, "VND"},
		{"missing currency", 1, 1, 2, 1000, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateDoubleEntry(
				nil,
				tt.transaction,
				tt.debit,
				tt.credit,
				tt.amount,
				tt.currency,
				0,
				0,
			); err == nil {
				t.Fatal("CreateDoubleEntry() expected validation error")
			}
		})
	}
}
