// Package arsipkan dipakai modul lain untuk memasukkan dokumen final ke arsip
// kepegawaian guru secara otomatis (tampil sebagai "Sistem · modul X").
package arsipkan

import (
	"time"

	"github.com/Caknoooo/go-gin-clean-starter/database/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Data struct {
	PendidikID     uuid.UUID
	SekolahID      uuid.UUID
	Kategori       string // entities.ArsipKategori*
	JenisDokumen   string
	Nomor          string
	TanggalDokumen *time.Time
	Keterangan     string
	DokumenID      uuid.UUID
	SumberModul    string // mis. "cuti"
	SumberID       uuid.UUID
	Oleh           uuid.UUID // akun yang memfinalkan dokumen
}

// Otomatis menulis arsip versi 1 beserta log unggah. Panggil di transaksi yang sama
// dengan langkah final modul supaya arsip dan status selalu konsisten.
func Otomatis(tx *gorm.DB, d Data) error {
	sumber := d.SumberID
	a := entities.Arsip{
		PendidikID: d.PendidikID, SekolahID: d.SekolahID, Kategori: d.Kategori,
		JenisDokumen: d.JenisDokumen, Nomor: d.Nomor, TanggalDokumen: d.TanggalDokumen,
		Keterangan: d.Keterangan, VersiTerkini: 1, SumberModul: d.SumberModul, SumberID: &sumber,
	}
	if err := tx.Omit("Pendidik", "Sekolah", "Versi").Create(&a).Error; err != nil {
		return err
	}
	if err := tx.Omit("Dokumen", "LampiranDokumen", "Pengunggah").Create(&entities.ArsipVersi{
		ArsipID: a.ID, Versi: 1, DokumenID: d.DokumenID, DiunggahOleh: d.Oleh, Catatan: "Diarsipkan otomatis",
	}).Error; err != nil {
		return err
	}
	return tx.Omit("Arsip", "User").Create(&entities.ArsipLog{
		ArsipID: a.ID, Versi: 1, Aksi: entities.ArsipAksiMengunggah, UserID: d.Oleh, Keterangan: "otomatis dari modul " + d.SumberModul,
	}).Error
}
