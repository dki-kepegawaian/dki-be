# Pensiun: catatan implementasi

## Endpoint (`/api/pensiun`)

| Metode | Path | Peran | Layar |
|---|---|---|---|
| GET | `` | semua (scope) | Q2, P6 |
| GET | `/ringkasan` | semua (scope) | Q2 kartu |
| GET | `/persyaratan` | semua | P1 "Lihat 14 berkas", P2 |
| GET | `/perkiraan?pendidik_id=` | semua (scope) | P1 |
| GET | `/:id` | semua (scope) | P3, P6 detail |
| POST | `/berkas` | operator_sekolah | P2 (satu berkas per request) |
| POST | `` | operator_sekolah | P2 Ajukan |
| POST | `/:id/perbaikan` | operator_sekolah | P6 Unggah ulang |
| POST | `/:id/verifikasi` | staf_sudin | P3 |
| POST | `/:id/kirim-bkd` | staf_sudin | P4 |
| POST | `/:id/sk` | staf_sudin | P5 |

Nomor registrasi: `PEN-YYYY-NNNN` (tahun pengajuan).

## Asumsi

- Hanya pendidik `PNS` yang bisa diajukan (usulan pensiun lewat BKD). PPPK ditolak 400.
- Satu pendidik hanya punya satu pengajuan pensiun, apa pun statusnya (pensiun tidak berulang). Tidak ada status "ditolak/dibatalkan" di Kamus Status.
- Batas usia pensiun guru 60 tahun. TMT `batas_usia` = tanggal 1 bulan sesudah ulang tahun ke-60 (Ahmad Fauzi lahir 5 Mar 1967 → 1 Apr 2027, sesuai P1). Butuh `tanggal_lahir` di data pendidik.
- `atas_permintaan_sendiri` (P6 contoh "1 Jan 2027 (atas permintaan sendiri)"): TMT diisi operator, harus tanggal 1 dan di masa depan. Syarat usia/masa kerja minimum APS tidak dicek.
- Masa kerja = TMT CPNS sampai TMT pensiun (tahun + bulan penuh); null bila `tmt_cpns` kosong. Angka di P1 ("38 tahun 8 bulan") tidak cocok dengan NIP contoh, jadi rumus ini asumsi.
- Berkas persyaratan: PDF saja, maks 2 MB (P2, state.txt). Unggah per berkas lewat `/berkas` karena 14 × 2 MB melebihi batas body multipart 11 MiB. Dokumen yang dirujuk harus diunggah akun yang sama, untuk sekolah pendidik, dan belum dipakai pengajuan pensiun lain.
- Semua 14 berkas wajib (termasuk akta nikah dan Karsu/Karis), sesuai "Lengkapi 14 berkas" di P2.
- Verifikasi: staf boleh mengirim penilaian sebagian; berkas yang tidak disebut mempertahankan penilaian lama, tetapi setelah digabung semua harus sudah dinilai. Sesudah perbaikan cukup menilai berkas pengganti.
- Perbaikan wajib mengganti persis semua berkas tidak valid, tidak boleh mengganti berkas valid.
- Pengiriman BKD per pengajuan (P4). Bukti tanda terima opsional (P4 hanya menandai nomor surat sebagai wajib). Tanggal kirim dan tanggal SK tidak boleh di masa depan.
- SK: `tmt_pensiun` opsional di P5; bila diisi menggantikan TMT tersimpan.
- Pimpinan dan admin hanya membaca; semua aksi Sudin oleh `staf_sudin` (mengikuti modul Cuti).

## Belum dibuat

- "Unduh 14 berkas (ZIP)" dan "Cetak daftar berkas" (P4), Bisa dibuat di frontend dari URL berkas di detail. "Unduh rekap" (Q2) memakai `GET /api/dasbor/pengajuan/ekspor?modul=pensiun` (CSV).
- "Simpan penilaian" sementara tanpa transisi (P3 hanya menampilkan "Tandai terverifikasi" / "Kembalikan untuk perbaikan").
- Dokumen yang diunggah lewat `/berkas` tetapi tidak pernah diajukan tetap tersimpan (tidak ada pembersihan berkala).

## Pertanyaan terbuka untuk pemilik produk

1. Apakah PPPK juga diajukan pensiun lewat Teralur? Bila ya, berkas dan alurnya sama?
2. Apakah pengajuan yang sudah `selesai` atau salah input perlu bisa dibatalkan/ditolak? Kamus Status tidak punya status itu.
3. BUP guru selalu 60 tahun? Rumus TMT (tanggal 1 bulan berikutnya) dan rumus masa kerja di P1 perlu dikonfirmasi.
4. Apakah akta nikah dan Karsu/Karis wajib untuk guru yang belum/tidak menikah?
5. Siapa yang mengisi `tanggal_lahir` dan `tmt_cpns` di data induk bila kosong (sekarang hanya staf Sudin/admin lewat `/api/pendidik`)?
