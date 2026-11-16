# Kinerja P3K Paruh Waktu: catatan implementasi

## Endpoint (`/api/kinerja-p3k`)

| Metode | Path | Peran | Fungsi |
|---|---|---|---|
| GET | `/api/kinerja-p3k` | semua (ter-scope) | Daftar penilaian; filter `status`, `tahun`, `semester`, `sekolah_id`, `search` |
| GET | `/api/kinerja-p3k/ringkasan` | semua | Jumlah per status, opsional per `tahun`/`semester` |
| GET | `/api/kinerja-p3k/pegawai` | semua | Pegawai `PPPK_PW` + penilaiannya untuk `tahun`+`semester` (KP1) |
| GET | `/api/kinerja-p3k/{id}` | semua | Detail + dokumen + riwayat |
| POST | `/api/kinerja-p3k` | operator_sekolah | Buat draf (`pendidik_id`, `tahun`, `semester`, `nilai`) |
| PUT | `/api/kinerja-p3k/{id}` | operator_sekolah | Ubah nilai draf |
| POST | `/api/kinerja-p3k/{id}/dokumen` | operator_sekolah | `draf` → `menunggu_ttd`, buat PDF |
| POST | `/api/kinerja-p3k/{id}/final` | staf_sudin | `menunggu_ttd` → `final` (multipart `berkas`, `ttd_jelas`, `data_sesuai`) |

Pimpinan dan admin hanya membaca. Data di luar scope menghasilkan 404. PDF diunduh lewat `GET /api/dokumen/{id}`.

## Asumsi

- Periode = semester (1: Januari–Juni, 2: Juli–Desember) per tahun, sesuai "Semester 2 · Juli–Desember 2026" di KP1. Tidak ada tabel periode; tahun 2020–2100, tanpa batas tenggat ("Nilai 2 pegawai sebelum 19 Desember 2026" belum dimodelkan).
- Skala 1–5 per parameter (KP2). Nilai akhir = rata-rata parameter terisi, dibulatkan 1 desimal (contoh Figma: 6 parameter → 4,2; 7 parameter → 4,3).
- Predikat dari nilai akhir: Sangat kurang < 1,5; Kurang 1,5–2,4; Cukup 2,5–3,4; Baik 3,5–4,4 (dari Figma); Sangat baik ≥ 4,5 (KP5 menampilkan 4,6 · Sangat baik). Rentang selain Baik diturunkan dengan lebar yang sama.
- Pejabat Penilai di PDF = "Kepala Sudin Pendidikan <nama Sudin sekolah>". Nama dan NIP pejabat dibiarkan kosong untuk ditulis tangan saat TTD basah; nomor wilayah ("Wilayah I") tidak tersimpan di data Sudin.
- Nomor penilaian `P3K-YYYY-NNNN` (dari layar Tugas Saya dan Arsip) diberikan saat draf dibuat; tahun nomor = tahun pembuatan, bukan tahun periode.
- Draf tidak memberi notifikasi; Sudin baru diberi tahu saat dokumen dibuat (menunggu TTD).
- Hanya operator yang mengisi nilai (KP1–KP3 di layar operator). Staf Sudin tidak bisa mengubah nilai.
- Pegawai PPPK_PW demo di fondasi ada di sekolah tanpa operator demo, jadi e2e membuat pegawai PPPK_PW sendiri di SMP Negeri 45 lewat `POST /api/pendidik` sebagai staf Sudin. Tidak ada seed tambahan.
- PDF memakai font inti Helvetica (cp1252). Karakter nama di luar cp1252 akan tampil tidak sempurna; cukup untuk nama Indonesia pada umumnya.

## Pertanyaan terbuka

1. Rentang predikat resmi selain "Baik (3,5–4,4)"? Apakah mengikuti Permenpan RB tentang penilaian kinerja PPPK?
2. Apakah dokumen yang sudah dibuat (menunggu TTD) bisa dikembalikan ke draf bila nilai salah, atau ditolak Sudin saat pindaian tidak sesuai? Saat ini tidak ada jalur balik.
3. Siapa pejabat penilai sebenarnya: Kepala Sudin (KP2) atau kepala sekolah (notifikasi di layar sekolah menyebut "siap ditandatangani kepala sekolah")?
4. Apakah ada tenggat per periode yang perlu disimpan (mis. 19 Desember) dan diingatkan lewat notifikasi?
5. Perlukah "Unduh rekap" KP4 sebagai berkas?
