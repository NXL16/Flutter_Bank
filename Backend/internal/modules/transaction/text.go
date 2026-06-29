package transaction

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func defaultTransferDescription(fullName string) string {
	name := removeVietnameseDiacritics(strings.TrimSpace(fullName))
	if name == "" {
		return "Chuyen khoan"
	}
	return name + " chuyen khoan"
}

func removeVietnameseDiacritics(value string) string {
	value = strings.ReplaceAll(value, "đ", "d")
	value = strings.ReplaceAll(value, "Đ", "D")

	var builder strings.Builder
	for _, char := range norm.NFD.String(value) {
		if unicode.Is(unicode.Mn, char) {
			continue
		}
		builder.WriteRune(char)
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}
