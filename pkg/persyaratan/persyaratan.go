// Package persyaratan berisi aturan berkas persyaratan yang sama untuk Pensiun dan
// Kenaikan Pangkat: unggah per berkas, ikat ke pengajuan, nilai per berkas, unggah ulang
// hanya berkas yang tidak valid. Tabel berkasnya tetap terpisah per modul.
package persyaratan

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/Caknoooo/go-gin-clean-starter/pkg/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MaxUkuran = 2 << 20 // 2 MB per berkas (Figma P2/K2)
	mimePDF   = "application/pdf"
)

// Penilaian satu berkas oleh Sudin (P3/K3).
type Penilaian struct {
	Jenis   string `json:"jenis" binding:"required,max=40"`
	Valid   *bool  `json:"valid" binding:"required"`
	Catatan string `json:"catatan" binding:"max=1000"`
}

// Penyimpan dipenuhi oleh dokumen service.
type Penyimpan interface {
	Simpan(ctx context.Context, tx *gorm.DB, fh *multipart.FileHeader, sekolahID *uuid.UUID, userID uuid.UUID) (entities.Dokumen, error)
	Hapus(docs ...entities.Dokumen)
}

func Label(daftar []entities.JenisBerkas, kode string) string {
	for _, j := range daftar {
		if j.Kode == kode {
			return j.Label
		}
	}
	return kode
}

// Unggah menyimpan satu berkas persyaratan (PDF, maks 2 MB) untuk sekolah operator.
// Berkas belum terikat ke pengajuan; ikat lewat Ambil saat mengajukan atau memperbaiki.
func Unggah(ctx context.Context, db *gorm.DB, p Penyimpan, actor entities.User, fh *multipart.FileHeader) (entities.Dokumen, error) {
	if actor.SekolahID == nil {
		return entities.Dokumen{}, utils.ErrBadRequest("akun tidak terhubung ke sekolah")
	}
	if fh != nil && fh.Size > MaxUkuran {
		return entities.Dokumen{}, utils.ErrBadRequest("ukuran berkas melebihi 2 MB; kompres lalu unggah ulang")
	}
	var doc entities.Dokumen
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d, err := p.Simpan(ctx, tx, fh, actor.SekolahID, actor.ID)
		if err != nil {
			return err
		}
		doc = d
		if d.MimeType != mimePDF {
			return utils.ErrBadRequest("format berkas harus PDF")
		}
		return nil
	})
	if err != nil {
		p.Hapus(doc)
		return entities.Dokumen{}, err
	}
	return doc, nil
}

// Lengkap memastikan semua jenis berkas wajib ada di unggahan.
func Lengkap(unggahan map[string]string, daftar []entities.JenisBerkas) error {
	var kurang []string
	for _, j := range daftar {
		if unggahan[j.Kode] == "" {
			kurang = append(kurang, j.Label)
		}
	}
	if len(kurang) > 0 {
		return utils.ErrBadRequest(fmt.Sprintf("lengkapi %d berkas: %s", len(kurang), strings.Join(kurang, "; ")))
	}
	return nil
}

// Ambil memvalidasi dokumen yang dirujuk unggahan (jenis -> dokumen_id): jenis dikenal,
// diunggah actor untuk sekolah yang sama, PDF, dan belum dipakai di tabelBerkas.
func Ambil(ctx context.Context, tx *gorm.DB, actor entities.User, sekolahID uuid.UUID, unggahan map[string]string,
	daftar []entities.JenisBerkas, tabelBerkas string) (map[string]entities.Dokumen, error) {
	if len(unggahan) == 0 {
		return nil, utils.ErrBadRequest("berkas wajib diisi")
	}
	kenal := map[string]bool{}
	for _, j := range daftar {
		kenal[j.Kode] = true
	}
	jenisDari := map[uuid.UUID]string{}
	ids := make([]uuid.UUID, 0, len(unggahan))
	for jenis, s := range unggahan {
		if !kenal[jenis] {
			return nil, utils.ErrBadRequest("jenis berkas tidak dikenal: " + jenis)
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, utils.ErrBadRequest("dokumen berkas " + jenis + " tidak valid")
		}
		if _, dup := jenisDari[id]; dup {
			return nil, utils.ErrBadRequest("satu dokumen tidak boleh dipakai untuk dua berkas")
		}
		jenisDari[id] = jenis
		ids = append(ids, id)
	}

	var docs []entities.Dokumen
	if err := tx.WithContext(ctx).Where("id IN ?", ids).Find(&docs).Error; err != nil {
		return nil, err
	}
	out := make(map[string]entities.Dokumen, len(docs))
	for _, d := range docs {
		if d.DiunggahOleh != actor.ID || d.SekolahID == nil || *d.SekolahID != sekolahID || d.MimeType != mimePDF {
			continue
		}
		out[jenisDari[d.ID]] = d
	}
	for _, j := range daftar {
		if _, diminta := unggahan[j.Kode]; diminta {
			if _, ok := out[j.Kode]; !ok {
				return nil, utils.ErrBadRequest("berkas " + j.Label + " tidak ditemukan; unggah ulang")
			}
		}
	}

	var dipakai int64
	if err := tx.WithContext(ctx).Table(tabelBerkas).Where("dokumen_id IN ?", ids).Count(&dipakai).Error; err != nil {
		return nil, err
	}
	if dipakai > 0 {
		return nil, utils.ErrConflict("berkas sudah dipakai di pengajuan lain; unggah ulang")
	}
	return out, nil
}

// Nilai menerapkan penilaian Sudin. Berkas yang tidak disebut mempertahankan nilai
// sebelumnya; setelahnya semua berkas wajib sudah dinilai. Mengembalikan berkas tidak valid.
func Nilai(berkas []*entities.BerkasPersyaratan, penilaian []Penilaian, oleh uuid.UUID, daftar []entities.JenisBerkas) ([]*entities.BerkasPersyaratan, error) {
	idx := make(map[string]*entities.BerkasPersyaratan, len(berkas))
	for _, b := range berkas {
		idx[b.Jenis] = b
	}
	now := time.Now()
	seen := map[string]bool{}
	for _, p := range penilaian {
		b, ok := idx[p.Jenis]
		if !ok {
			return nil, utils.ErrBadRequest("berkas " + p.Jenis + " tidak ada di pengajuan ini")
		}
		if seen[p.Jenis] {
			return nil, utils.ErrBadRequest("berkas " + p.Jenis + " dinilai lebih dari sekali")
		}
		seen[p.Jenis] = true
		catatan := strings.TrimSpace(p.Catatan)
		if !*p.Valid && catatan == "" {
			return nil, utils.ErrBadRequest("alasan tidak valid wajib diisi untuk " + Label(daftar, p.Jenis))
		}
		if *p.Valid {
			catatan = ""
		}
		v := *p.Valid
		b.Valid, b.Catatan, b.DinilaiOleh, b.DinilaiAt = &v, catatan, &oleh, &now
	}

	var tidakValid []*entities.BerkasPersyaratan
	for _, b := range berkas {
		if b.Valid == nil {
			return nil, utils.ErrBadRequest("berkas " + Label(daftar, b.Jenis) + " belum dinilai")
		}
		if !*b.Valid {
			tidakValid = append(tidakValid, b)
		}
	}
	return tidakValid, nil
}

// RingkasTidakValid menyusun "Label: catatan; ..." untuk riwayat dan notifikasi.
func RingkasTidakValid(tidakValid []*entities.BerkasPersyaratan, daftar []entities.JenisBerkas) string {
	parts := make([]string, 0, len(tidakValid))
	for _, b := range tidakValid {
		parts = append(parts, Label(daftar, b.Jenis)+": "+b.Catatan)
	}
	return strings.Join(parts, "; ")
}

// Ganti memasang dokumen baru untuk berkas yang dinyatakan tidak valid. Unggahan harus
// mencakup persis semua berkas tidak valid; berkas yang valid tidak boleh diganti.
func Ganti(berkas []*entities.BerkasPersyaratan, dok map[string]entities.Dokumen, daftar []entities.JenisBerkas) error {
	for _, b := range berkas {
		d, ada := dok[b.Jenis]
		tidakValid := b.Valid != nil && !*b.Valid
		switch {
		case tidakValid && !ada:
			return utils.ErrBadRequest("berkas " + Label(daftar, b.Jenis) + " tidak valid dan wajib diunggah ulang")
		case !tidakValid && ada:
			return utils.ErrBadRequest("berkas " + Label(daftar, b.Jenis) + " tidak perlu diunggah ulang")
		case ada:
			b.DokumenID = d.ID
			b.Valid, b.Catatan, b.DinilaiOleh, b.DinilaiAt = nil, "", nil, nil
		}
	}
	return nil
}

// DokumenRingkas adalah metadata berkas untuk respons (tanpa path penyimpanan).
type DokumenRingkas struct {
	ID       string `json:"id"`
	NamaAsli string `json:"nama_asli"`
	MimeType string `json:"mime_type"`
	Ukuran   int64  `json:"ukuran"`
	URL      string `json:"url"`
}

func Dokumen(d *entities.Dokumen) *DokumenRingkas {
	if d == nil {
		return nil
	}
	return &DokumenRingkas{
		ID: d.ID.String(), NamaAsli: d.NamaAsli, MimeType: d.MimeType, Ukuran: d.Ukuran,
		URL: "/api/dokumen/" + d.ID.String(),
	}
}

type BerkasResponse struct {
	Jenis     string          `json:"jenis"`
	Label     string          `json:"label"`
	Kelompok  string          `json:"kelompok"`
	Dokumen   *DokumenRingkas `json:"dokumen"`
	Valid     *bool           `json:"valid"`
	Catatan   string          `json:"catatan"`
	DinilaiAt *time.Time      `json:"dinilai_at"`
}

// Item adalah satu berkas beserta dokumennya (relasi Dokumen ada di struct tabel modul).
type Item struct {
	Berkas  entities.BerkasPersyaratan
	Dokumen *entities.Dokumen
}

// KeResponse mengurutkan berkas mengikuti daftar wajib.
func KeResponse(items []Item, daftar []entities.JenisBerkas) []BerkasResponse {
	idx := make(map[string]Item, len(items))
	for _, it := range items {
		idx[it.Berkas.Jenis] = it
	}
	out := make([]BerkasResponse, 0, len(items))
	for _, j := range daftar {
		it, ok := idx[j.Kode]
		if !ok {
			continue
		}
		out = append(out, BerkasResponse{
			Jenis: j.Kode, Label: j.Label, Kelompok: j.Kelompok, Dokumen: Dokumen(it.Dokumen),
			Valid: it.Berkas.Valid, Catatan: it.Berkas.Catatan, DinilaiAt: it.Berkas.DinilaiAt,
		})
	}
	return out
}

// MasaKerja menghitung selisih tahun dan bulan penuh antara dua tanggal.
func MasaKerja(dari, sampai time.Time) (tahun, bulan int) {
	n := (sampai.Year()-dari.Year())*12 + int(sampai.Month()-dari.Month())
	if sampai.Day() < dari.Day() {
		n--
	}
	if n < 0 {
		n = 0
	}
	return n / 12, n % 12
}
