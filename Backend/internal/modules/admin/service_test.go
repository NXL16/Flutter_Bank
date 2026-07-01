package admin

import "testing"

func TestValidateAdminPassword(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		shouldFail bool
	}{
		{name: "valid", password: "StrongAdmin12!", shouldFail: false},
		{name: "too short", password: "Admin1!", shouldFail: true},
		{name: "missing uppercase", password: "strongadmin12!", shouldFail: true},
		{name: "missing lowercase", password: "STRONGADMIN12!", shouldFail: true},
		{name: "missing number", password: "StrongAdmin!!", shouldFail: true},
		{name: "missing special", password: "StrongAdmin123", shouldFail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAdminPassword(tt.password)
			if tt.shouldFail && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.shouldFail && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestNormalizePhone(t *testing.T) {
	for input, expected := range map[string]string{
		"0901234567":     "901234567",
		"+84901234567":   "901234567",
		"84 901 234 567": "901234567",
	} {
		if actual := normalizePhone(input); actual != expected {
			t.Fatalf("normalizePhone(%q) = %q, want %q", input, actual, expected)
		}
	}
}
