package transaction

import "testing"

func TestDefaultTransferDescription(t *testing.T) {
	tests := map[string]string{
		"Nguyễn Xuân Linh": "Nguyen Xuan Linh chuyen khoan",
		"Đào Thành Nhân":   "Dao Thanh Nhan chuyen khoan",
		"  Trần   Thị B  ": "Tran Thi B chuyen khoan",
		"":                 "Chuyen khoan",
	}

	for input, expected := range tests {
		if actual := defaultTransferDescription(input); actual != expected {
			t.Errorf("defaultTransferDescription(%q) = %q, want %q", input, actual, expected)
		}
	}
}
