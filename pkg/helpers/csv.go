package helpers

import "strings"

// AmanCSV mencegah CSV/formula injection saat berkas dibuka di spreadsheet:
// nilai yang diawali = + - @ tab atau CR diberi awalan tanda petik.
func AmanCSV(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
