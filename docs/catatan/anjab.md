# Analisis Jabatan (Anjab): catatan implementasi

## Endpoint (`/api/anjab`)

| Metode | Path | Peran | Layar |
|---|---|---|---|
| GET | `/periode` | semua (scope Sudin) | A1 |
| POST | `/periode` | staf_sudin | A1 |
| GET | `/periode/:id` | semua (scope Sudin) | A1, A3 (progres) |
| PUT | `/periode/:id` | staf_sudin, sebelum dibuka | A1 |
| GET | `/periode/:id/templat` | staf_sudin | A2 |
| POST | `/periode/:id/impor` | staf_sudin, sebelum dibuka (`simpan=true` untuk menyimpan) | A2 |
| POST | `/periode/:id/buka` | staf_sudin | A3 |
| PUT | `/periode/:id/tenggat` | staf_sudin, setelah dibuka | A3 Perpanjang tenggat |
| POST | `/periode/:id/pengingat` | staf_sudin, setelah dibuka | A3 Kirim pengingat |
| GET | `/periode/:id/monitoring?mapel=` | staf_sudin, pimpinan, admin | A6 |
| GET | `/periode/:id/monitoring/excel` | staf_sudin, pimpinan, admin | A6 Unduh Excel |
| GET | `/sekolah?periode_id=&status=&kecamatan=&search=` | semua (scope sekolah) | A3 daftar, A4 |
| GET | `/sekolah/:id` | semua (scope sekolah) | A4, A5 |
| PUT | `/sekolah/:id/rombel` | operator_sekolah | A4 |
| POST | `/sekolah/:id/guru` | operator_sekolah | A4 Tambah guru |
| PUT | `/sekolah/:id/guru/:guruId` | operator_sekolah | A4 mapel/JP riil |
| DELETE | `/sekolah/:id/guru/:guruId` | operator_sekolah | A4 |
| POST | `/sekolah/:id/kirim` | operator_sekolah | A4 Kirim pembaruan |
| POST | `/sekolah/:id/tinjau` | staf_sudin | A5 Tetapkan / Kembalikan |

Status per sekolah: `draf` → `dikirim` → (`dikembalikan` → `dikirim`)* → `ditetapkan` (final).
Setiap transisi: transaksi, `SELECT ... FOR UPDATE` dalam scope, cek status pendahulu persis (409), `anjab_riwayat`,
`aktivitas.Catat`, notifikasi lewat `pkg/penerima`.

## Rumus (fungsi murni `modules/anjab/hitung`, diuji di `hitung_test.go`)

- `rombel_dihitung = min(jumlah_rombel, ceil(jumlah_siswa / rasio_jenjang))`; tanpa data siswa dipakai jumlah rombel apa adanya.
- Guru mapel: `kebutuhan = ceil(Σ_tingkat rombel_dihitung × JP mapel di tingkat itu / jam_wajib)`.
- SD guru kelas (mapel `Guru Kelas`): `kebutuhan = Σ rombel_dihitung`.
- Tersedia = jumlah guru di daftar sekolah dengan mapel itu, kecuali guru yang TMT pensiunnya ≤ `batas_tmt_pensiun`.
  TMT pensiun = tanggal 1 bulan berikutnya setelah ulang tahun ke-60 (dari `pendidik.tanggal_lahir`). Tanpa tanggal lahir tetap dihitung.
- **Dua basis** (A5 "Versi SK" dan "Versi kondisi riil", G01/G02 "dihitung di 2 basis"): versi SK memakai `mapel_sk`,
  versi riil memakai `mapel_riil` (bila kosong jatuh ke `mapel_sk`). Selisih = tersedia − kebutuhan; total kurang/lebih per versi.
  `perbedaan` mendaftar guru yang mapel riilnya berbeda dari SK ("Perbedaan versi berasal dari 1 guru").
- Mapel yang ada di data guru tapi tidak punya beban JP tetap ditampilkan (kebutuhan 0, jadi kelebihan).
- Nama mapel dicocokkan tanpa peka huruf besar/kecil dan spasi ganda.
- Rekomendasi pemerataan (A6): per mapel, sekolah dengan kekurangan terbesar dilayani dulu; sumber diutamakan dari
  kecamatan yang sama, lalu kelebihan terbesar. Hanya sekolah berstatus ditetapkan, versi riil.

## Asumsi

1. Periode milik satu Sudin, unik per tahun ajaran. Admin hanya membaca (tidak punya Sudin untuk dijadikan pemilik).
2. Rasio siswa per rombel di A1 dipakai sebagai batas atas rombel yang dihitung (mencegah rombel kecil-kecil menaikkan
   kebutuhan). Brief awal hanya menyebut "jumlah rombel"; bila rasio cukup sebagai peringatan, ganti `RombelDihitung`.
3. Guru tersedia dihitung per kepala (1 guru = 1), bukan dari JP riil. JP riil disimpan dan ditampilkan saja.
4. Satu mapel riil per guru. H3 menampilkan "Riil: Informatika, Prakarya" (dua mapel); belum didukung.
5. "Guru dengan pensiun aktif tidak dihitung" ditafsirkan hanya lewat TMT pensiun ≤ batas; belum membaca status
   usulan di modul Pensiun.
6. Membuka masa pembaruan tidak menunggu `tanggal_buka` (tombol eksplisit di A3); ditolak bila `tanggal_tutup` sudah lewat.
   Operator hanya bisa mengubah/mengirim sampai `tanggal_tutup` (WIB). Peninjauan Sudin tidak dibatasi tenggat.
7. Impor hanya sebelum masa pembaruan dibuka. Data rombel/guru sekolah yang muncul di berkas menggantikan data
   periode itu; sekolah yang tidak muncul tidak disentuh. Validasi dan simpan memakai unggahan yang sama dua kali
   (stateless), sesuai tombol "Impor ulang" dan "Simpan N baris valid" di A2.
8. Lembar Guru: NIP wajib untuk PNS/PPPK (18 digit, spasi dibuang), NIK wajib 16 digit (Figma menandai NIK kosong
   sebagai masalah), status `PNS/PPPK/PPPK_PW/KKI` (juga "PPPK Paruh Waktu"), tanggal YYYY-MM-DD, DD/MM/YYYY, atau sel
   tanggal Excel. NIP yang sudah terdaftar di sekolah luar Sudin ditolak per baris. Guru tanpa NIP dicocokkan berdasar nama
   di sekolah yang sama.
9. Upsert pendidik dari impor memperbarui sekolah, nama, NIK, status kepegawaian, dan (bila diisi) golongan, tanggal lahir,
   TMT CPNS. Kolom pendidik lain tidak disentuh. Guru baru diberi jabatan "Guru Mapel" atau "Guru Kelas".
10. Operator menambah guru hanya dari pendidik yang sudah terdaftar di sekolahnya; mapel SK hanya diisi saat tambah,
    selanjutnya operator mengubah mapel riil dan JP riil saja (H3: perubahan identitas lewat Sudin).
11. Kirim mensyaratkan ada rombel, ada guru, dan semua guru yang dihitung sudah punya mapel riil (H3 "data perlu dilengkapi").
12. Berkas .xlsx impor tidak disimpan (modul dokumen hanya menerima PDF/JPG/PNG); hanya ringkasan di log aktivitas.

## Belum dibuat / pertanyaan terbuka

- A6 "Unduh laporan Anjab (PDF)" ada di `/periode/:id/monitoring/pdf`; rekap Excel di `/monitoring/excel`.
- A2 "Unduh daftar galat" dan A3 "Unduh rekap" belum punya endpoint. Daftar galat sudah lengkap di respons impor
  (berkas impor tidak disimpan), jadi berkas unduhannya dibentuk di frontend.
- A6 "jarak" antar-sekolah: belum ada data koordinat; hanya `kecamatan_sama` (true/false). Figma menyebut "Berdekatan".
- Apakah rasio siswa per rombel memang membatasi rombel yang dihitung, atau hanya peringatan?
- Apakah kebutuhan memakai JP riil guru (FTE) alih-alih jumlah kepala?
- Guru dengan dua mapel riil: dihitung di mapel mana?
- Apakah "pensiun aktif" juga mencakup guru yang usulan pensiunnya sedang diproses (modul Pensiun)?
- Layar G01/G02 tidak ada di berkas teks Figma yang tersedia; tafsiran "2 basis" diambil dari A5.
- Perlu kah Pimpinan ikut menetapkan (A5 hanya menyebut staf Sudin)?
