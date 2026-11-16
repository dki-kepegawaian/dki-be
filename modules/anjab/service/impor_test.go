package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buku membuat .xlsx satu lembar "Sekolah" lewat StreamWriter (hemat memori di sisi tes).
func buku(t *testing.T, isi func(sw *excelize.StreamWriter)) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if _, err := f.NewSheet(lembarSekolah); err != nil {
		t.Fatal(err)
	}
	sw, err := f.NewStreamWriter(lembarSekolah)
	if err != nil {
		t.Fatal(err)
	}
	header := make([]any, len(kolomSekolah))
	for i, k := range kolomSekolah {
		header[i] = k
	}
	if err := sw.SetRow("A1", header); err != nil {
		t.Fatal(err)
	}
	isi(sw)
	if err := sw.Flush(); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	out, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	return out
}

func TestBacaLembarBiasa(t *testing.T) {
	f := buku(t, func(sw *excelize.StreamWriter) {
		_ = sw.SetRow("A2", []any{"20101045", " SMP Negeri 45 ", "SMP", "Kebon Jeruk"})
		_ = sw.SetRow("A4", []any{"20101436", "SD Negeri Rawa Buaya 09", "SD", "Cengkareng"})
	})
	l, err := bacaLembar(f, lembarSekolah, kolomSekolah)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.baris) != 2 || l.nomor[0] != 2 || l.nomor[1] != 4 {
		t.Fatalf("baris=%d nomor=%v, mau 2 baris bernomor 2 dan 4", len(l.baris), l.nomor)
	}
	if got := l.sel(l.baris[0], "Nama Sekolah"); got != "SMP Negeri 45" {
		t.Fatalf("Nama Sekolah = %q", got)
	}
}

// Sel di kolom terakhir (XFD) membuat excelize mengisi baris sampai 16.384 elemen.
// Yang disimpan harus tetap selebar kolom templat saja.
func TestBacaLembarSelKolomJauhTidakDisimpan(t *testing.T) {
	f := buku(t, func(sw *excelize.StreamWriter) {
		lebar := make([]any, 16384)
		lebar[0], lebar[1], lebar[2], lebar[3], lebar[16383] = "20101045", "SMP Negeri 45", "SMP", "Kebon Jeruk", 1
		_ = sw.SetRow("A2", lebar)
		for i := 3; i <= 300; i++ { // baris yang hanya berisi sel XFD: dianggap kosong
			c, _ := excelize.CoordinatesToCellName(16384, i)
			_ = sw.SetRow(c, []any{1})
		}
	})
	l, err := bacaLembar(f, lembarSekolah, kolomSekolah)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.baris) != 1 {
		t.Fatalf("baris tersimpan = %d, mau 1", len(l.baris))
	}
	if n := cap(l.baris[0]); n > len(kolomSekolah) {
		t.Fatalf("kapasitas baris tersimpan = %d, mau <= %d", n, len(kolomSekolah))
	}
}

func TestBacaLembarBerhentiDiBatasBaris(t *testing.T) {
	f := buku(t, func(sw *excelize.StreamWriter) {
		for i := 2; i <= maksBarisImpor+50; i++ {
			c, _ := excelize.CoordinatesToCellName(1, i)
			_ = sw.SetRow(c, []any{"x"})
		}
	})
	_, err := bacaLembar(f, lembarSekolah, kolomSekolah)
	if err == nil || !strings.Contains(err.Error(), "maksimal") {
		t.Fatalf("err = %v, mau galat batas baris", err)
	}
}
