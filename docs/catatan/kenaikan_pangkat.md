# Kenaikan Pangkat: catatan implementasi

## Endpoint (`/api/kenaikan-pangkat`)

| Metode | Path | Peran | Layar |
|---|---|---|---|
| GET | `` (filter `status`, `periode`, `sekolah_id`, `search`) | semua (scope) | K1, K6, Q3, K4 antrean |
| GET | `/ringkasan` | semua (scope) | Q3 kartu |
| GET | `/persyaratan` | semua | K1 "Lihat persyaratan berkas", K2 |
| GET | `/perkiraan?pendidik_id=` | semua (scope) | K2 "Cari data" |
| GET | `/:id` | semua (scope) | K3, K6 detail |
| POST | `/berkas` | operator_sekolah | K2 (satu berkas per request) |
| POST | `` | operator_sekolah | K2 Ajukan |
| POST | `/:id/perbaikan` | operator_sekolah | K6 Perbaiki berkas |
| POST | `/:id/verifikasi` | staf_sudin | K3 |
| POST | `/kirim-bkd` | staf_sudin | K4 (batch) |
| POST | `/:id/sk` | staf_sudin | K5 |

Nomor registrasi: prefix **`KP`** → `KP-YYYY-NNNN` (tahun pengajuan), sesuai contoh di K1/K3/Q3.

## Asumsi

- Hanya pendidik `PNS` dengan golongan ruang dikenal (I/a … IV/e). Usulan = naik satu tingkat golongan; nama pangkat dari tabel PNS standar (`entities.PangkatPNS`). Bila `pangkat` di data induk kosong, pangkat lama diambil dari tabel. IV/e ditolak.
- Satu pendidik hanya boleh punya satu usulan yang belum `selesai`. Setelah selesai boleh diusulkan lagi ke golongan berikutnya.
- Periode (K1: "Periode April 2027", batas 31 Jan): usul s.d. akhir Januari → 1 April tahun itu; Februari–Juli → 1 Oktober; Agustus–Desember → 1 April tahun berikutnya. Q3 menyebut batas 15 Januari; dipakai akhir bulan.
- Syarat "4 tahun di golongan sekarang" (K1) dicek terhadap tanggal periode memakai `pendidik.tmt_golongan`. Bila kolom itu kosong, syarat tidak dicek (pengajuan tetap bisa dibuat) dan guru tidak muncul di `GET /api/kenaikan-pangkat/memenuhi-syarat`. `tmt_golongan` diperbarui otomatis ke TMT pangkat saat SK dicatat (K5).
- Berkas: 6 berkas K2, semuanya wajib, PDF maks 2 MB. Q3 menyebut "berkas 9/9" dan jenis Reguler/Fungsional; diabaikan, mengikuti K2.
- Verifikasi dan perbaikan sama dengan Pensiun (penilaian per berkas, perbaikan persis berkas yang tidak valid).
- K4 batch: semua usulan harus `terverifikasi` dan dalam wilayah staf; satu gagal → semua batal. Bukti opsional, disimpan satu salinan per sekolah supaya operator tiap sekolah bisa mengunduh (dokumen di-scope per sekolah).
- K5: `tmt_pangkat` wajib ("Mengikuti TMT yang tertulis di SK"). Golongan pendidik diperbarui hanya jika masih sama dengan golongan lama usulan; bila sudah sama dengan golongan baru dianggap sudah diperbarui; selain itu 409. Pendidik dikunci berdasarkan id tanpa scope (usulannya sudah dicek scope), jadi pembaruan tetap berjalan walau guru pindah sekolah.
- "TMT golongan" di K5 (sebelum → sesudah) tidak disimpan di data induk karena kolomnya tidak ada; TMT tersimpan di usulan (`tmt_pangkat`).
- "Simpan draf" (K2) dipenuhi dengan unggah per berkas: id dokumen bisa disimpan di frontend sampai Ajukan. Tidak ada status draf karena Kamus Status tidak memuatnya.

## Belum dibuat

- "Unduh daftar usulan" (K4), "Buka arsip guru" (K6).
- Pembersihan dokumen yang diunggah tetapi tidak pernah diajukan.

## Pertanyaan terbuka untuk pemilik produk

1. `tmt_golongan` sudah ada di data pendidik, tetapi masih kosong untuk data lama. Siapa yang mengisinya (impor dari BKD, atau staf satu per satu)? Selama kosong, syarat 4 tahun tidak dicek.
2. Berkas KP 6 (K2) atau 9 (Q3)? Apakah berkas berbeda untuk KP reguler vs fungsional?
3. Batas usul sekolah 31 Januari (K1) atau 15 Januari (Q3)? Dan batas untuk periode Oktober?
4. Apakah kenaikan bisa lompat lebih dari satu golongan (mis. penyesuaian ijazah)? Sekarang selalu satu tingkat.
5. Apakah usulan yang ditolak BKD perlu status sendiri? Kamus Status tidak punya.
