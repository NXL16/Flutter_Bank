package transaction

import "testing"

func TestValidateTransactionPIN(t *testing.T) {
	for _, pin := range []string{"482951", "907314"} {
		if err := validateTransactionPIN(pin); err != nil {
			t.Errorf("validateTransactionPIN(%q) unexpected error: %v", pin, err)
		}
	}
	for _, pin := range []string{"123456", "111111", "12345", "abcdef"} {
		if err := validateTransactionPIN(pin); err == nil {
			t.Errorf("validateTransactionPIN(%q) expected error", pin)
		}
	}
}
