package service

import "testing"

func TestAmanMenetralkanFormula(t *testing.T) {
	for _, s := range []string{"=1+1", "+1", "-1", "@SUM(A1)", "\t=1+1", "\r=1+1"} {
		if got := aman(s); got != "'"+s {
			t.Errorf("aman(%q) = %q, mau diawali tanda petik", s, got)
		}
	}
	for _, s := range []string{"", "SK Pensiun", "800/123", "a=b"} {
		if got := aman(s); got != s {
			t.Errorf("aman(%q) = %q, mau tidak berubah", s, got)
		}
	}
}
