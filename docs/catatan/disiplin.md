# Modul Sikap Disiplin: catatan implementasi

Paket `modules/disiplin`, route `/api/disiplin`, nomor registrasi `DSP-YYYY-NNNN`.
Migrasi `20261115030000_create_disiplin_tables` (tabel `disiplin`, `disiplin_berkas`, `disiplin_riwayat`).

## Endpoint

| Metode | Path | Peran | Figma | Transisi |
|---|---|---|---|---|
| GET | `/api/disiplin` | semua (scope) | DS9, DS8, B1 | filter `status` (koma), `sekolah_id`, `tahun`, `search`, paginasi |
| GET | `/api/disiplin/ringkasan` | semua (scope) | DS9, B2 | jumlah per status, perlu tindakan, menunggu keputusan, tenggat ≤ 5 hari kerja |
| GET | `/api/disiplin/{id}` | semua (scope) | semua layar | isi disaring per peran |
| GET | `/api/disiplin/{id}/dokumen/{dokumen_id}` | semua (scope) | "Lihat" | operator: hanya berkas non-internal |
| POST | `/api/disiplin` | operator | DS1 | → `registrasi` |
| POST | `/api/disiplin/{id}/verifikasi` | staf | DS2 | `registrasi` → `pemeriksaan` / `perlu_kelengkapan` / `tidak_dilanjutkan` |
| POST | `/api/disiplin/{id}/kelengkapan` | operator | DS3 | `perlu_kelengkapan` → `registrasi` |
| POST | `/api/disiplin/{id}/pemeriksaan` | staf | DS4 | tetap `pemeriksaan` (simpan berulang) |
| POST | `/api/disiplin/{id}/rekomendasi` | staf | DS5 | `pemeriksaan` → `menunggu_keputusan` |
| POST | `/api/disiplin/{id}/keputusan` | **pimpinan saja** | DS6 | `menunggu_keputusan` → `sanksi_ditetapkan` / `pemeriksaan` (kembalikan) |
| POST | `/api/disiplin/{id}/catat-keputusan` | staf | DS7 | `sanksi_ditetapkan` → `tindak_lanjut` |
| POST | `/api/disiplin/{id}/tindak-lanjut` | staf | DS7 | tetap `tindak_lanjut` |
| POST | `/api/disiplin/{id}/laporan-pelaksanaan` | operator | DS8 | tetap `tindak_lanjut` |
| POST | `/api/disiplin/{id}/tutup` | staf | DS7 | `tindak_lanjut` → `selesai` |

Semua transisi: transaksi, `SELECT ... FOR UPDATE` dalam scope, cek status pendahulu persis (409), `disiplin_riwayat`,
`aktivitas.Catat`, notifikasi lewat `pkg/penerima`. Simpan tanpa transisi (pemeriksaan, tindak lanjut, laporan
pelaksanaan) tetap dikunci dan dicek statusnya, dicatat di `log_aktivitas` saja (bukan riwayat status).
`selesai` dan `tidak_dilanjutkan` final.

## Keputusan kerahasiaan (apa yang dilihat siapa)

- **Operator sekolah** hanya kasus sekolahnya. Yang terlihat: laporan sendiri (uraian, kronologi, berkas yang diunggah
  sekolah), hasil verifikasi (keputusan, daftar kekurangan/alasan tidak layak, batas kelengkapan, tanpa butir daftar periksa
  internal), status, tahap, tenggat, hukuman yang ditetapkan (tingkat, jenis, nomor/tanggal/masa keputusan, salinan
  keputusan), laporan pelaksanaannya, dan kontak petugas Sudin (nama + email, sesuai DS8).
- **Tidak terlihat oleh operator** (DS8: "Rincian pemeriksaan dan BAP hanya untuk Sudin"): surat panggilan, waktu dan
  ringkasan pemeriksaan, BAP, tanggapan tertulis pegawai, bukti tambahan, rekomendasi (fakta, dasar, usulan hukuman,
  justifikasi), keputusan dan catatan pimpinan, bukti penyampaian, status I'DIS, upaya administratif, riwayat hukuman
  pegawai, dan `catatan` riwayat status selain transisi yang memang ditujukan ke sekolah (perlu kelengkapan, tidak
  dilanjutkan, kelengkapan). Alasannya: usulan hukuman yang belum/berbeda dengan keputusan, keterangan pegawai saat
  pemeriksaan, dan riwayat hukuman adalah data pribadi pegawai yang tidak diperlukan sekolah untuk menjalankan tugasnya.
- **Berkas internal** disimpan dengan `dokumen.sekolah_id = NULL` (menyimpang dari pola `Simpan(&sekolahID)` secara
  sengaja) supaya tidak bisa diunduh operator lewat `/api/dokumen/{id}` walau UUID-nya bocor. Semua berkas kasus diunduh
  lewat `/api/disiplin/{id}/dokumen/{dokumen_id}` yang mengecek scope kasus, kepemilikan berkas, dan flag internal.
  Service modul membaca object storage langsung (`pkg/storage` disuntik dari `providers`).
- **Pimpinan**: daftar kasus (`GET /api/disiplin`) dikirim tanpa `pendidik` mengikuti catatan B1 "Identitas guru hanya
  tampil di halaman kasus untuk pejabat berwenang"; detail kasus lengkap.
- **Notifikasi** hanya memuat nomor kasus dan nama sekolah, tanpa nama pegawai.
- **Riwayat hukuman** (DS6 "Riwayat hukuman") hanya kasus lain pegawai yang sama dalam scope pembaca (Sudin-nya).

## Asumsi

1. **Jenis pelanggaran**: Figma hanya memuat "Ketidakhadiran tanpa keterangan". Daftar ringkas dari kewajiban/larangan
   PP 94/2021: `ketidakhadiran`, `jam_kerja`, `target_kinerja`, `penyalahgunaan_wewenang`, `gratifikasi`, `netralitas`,
   `barang_milik_negara`, `lainnya`.
2. **Jenis hukuman** mengikuti PP 94/2021 Pasal 8: ringan (teguran lisan, teguran tertulis, pernyataan tidak puas
   tertulis), sedang (potong tukin 25% selama 6/9/12 bulan), berat (penurunan jabatan setingkat 12 bulan, pembebasan
   jabatan menjadi pelaksana 12 bulan, pemberhentian dengan hormat tidak atas permintaan sendiri). `jenis_hukuman` harus
   cocok dengan `tingkat` (400). Contoh di B1 ("Penundaan kenaikan gaji berkala 1 tahun") berasal dari PP 53/2010 yang
   sudah dicabut, jadi tidak dipakai.
3. **Masa hukuman** dihitung otomatis dari `masa_mulai` + lama hukuman − 1 hari (6 bulan dari 1 Des 2026 → 31 Mei 2027,
   sesuai DS7). Hukuman tanpa masa (ringan, pemberhentian) tidak menyimpan masa. Masa tidak boleh mulai sebelum tanggal keputusan.
4. **Status kepegawaian**: pegawai KKI ditolak (400) karena PP 94/2021 dan disiplin PPPK berlaku untuk ASN; PNS, PPPK,
   PPPK_PW diterima. Pencatatan I'DIS BKN hanya untuk PNS (DS7 "Khusus PNS") dan wajib sebelum tutup kasus PNS.
5. **Tenggat per tahap** (hari kerja, Senin–Jumat, libur nasional belum dihitung): verifikasi 5, kelengkapan 10 (bisa
   diisi staf lewat `batas_kelengkapan`), pemeriksaan + rekomendasi 20, keputusan pimpinan 5, pencatatan keputusan 5,
   laporan pelaksanaan 7. Angka diambil dari contoh layar yang kira-kira cocok, bukan dari regulasi.
6. **Verifikasi `layak`** hanya bila keempat butir daftar periksa `true`. `perlu_kelengkapan` dan `tidak_layak` wajib `catatan`.
7. **Pemeriksaan (DS4)** boleh disimpan berulang; surat panggilan bisa dicatat sebelum pemeriksaan, BAP menyusul.
   Rekomendasi (DS5) baru bisa diajukan bila ringkasan pemeriksaan, BAP, dan tanggapan tertulis pegawai sudah ada
   (DS4: "Wajib sebelum rekomendasi"). Berkas baru ditambahkan, tidak menggantikan yang lama.
8. **Keputusan pimpinan (DS6)**: `setujui` memakai hukuman rekomendasi; `ubah` wajib tingkat + jenis + catatan;
   `kembalikan` wajib catatan dan status kembali ke `pemeriksaan` (rekomendasi lama ditimpa saat diajukan ulang).
   Sekolah diberi tahu saat sanksi ditetapkan (konfirmasi DS6) dan melihat tingkat/jenis hukuman sejak itu.
9. **Tutup kasus** mensyaratkan: bukti penyampaian, laporan pelaksanaan sekolah, upaya administratif diisi (bila `ada`,
   `catatan_upaya` wajib), I'DIS untuk PNS. Proses upaya administratif sendiri di luar sistem.
10. **Berkas**: aturan unggah dari modul dokumen (PDF/JPG/PNG, maks 10 MB), bukan "PDF maks 2 MB" di Figma.
11. **Waktu pemeriksaan** dikirim `YYYY-MM-DDTHH:MM` dan dibaca sebagai WIB.
12. "Unduh rekap" (DS9) memakai `GET /api/dasbor/pengajuan/ekspor?modul=disiplin` (CSV; kolom pegawai kosong untuk
    pimpinan). "Buat dari Generator Surat" (DS4) belum dibuat.

## Pertanyaan terbuka untuk pemilik produk

1. Apakah daftar jenis pelanggaran dan katalog hukuman di atas sudah cukup, atau perlu daftar resmi Dinas (mis. pelanggaran
   khusus guru seperti kekerasan terhadap peserta didik)?
2. Apakah PPPK memakai katalog hukuman yang sama (PP 94/2021) atau aturan disiplin PPPK tersendiri? Apakah KKI perlu alur
   pembinaan terpisah?
3. Siapa yang menyampaikan keputusan ke pegawai: Sudin atau Kepala Sekolah? (DS7 menyimpan bukti penyampaian di Sudin,
   DS8 menunjukkan sekolah yang menyampaikan.) Saat ini bukti penyampaian diunggah staf dan bersifat internal.
4. Bila pegawai mengajukan upaya administratif (`ada`), apakah kasus harus menunggu hasilnya sebelum ditutup, dan apakah
   hukuman bisa berubah? Saat ini cukup dicatat di `catatan_upaya`.
5. Apakah catatan pimpinan (mis. "disertai pembinaan oleh Kepala Sekolah") perlu diteruskan ke sekolah? Saat ini hanya
   untuk Sudin; staf bisa menyampaikannya lewat jalur lain.
6. Tenggat resmi per tahap dan perhitungan hari libur nasional.
7. Apakah riwayat hukuman pegawai yang pindah Sudin perlu terlihat lintas wilayah (saat ini dibatasi scope pembaca)?
8. Pengingat akhir masa hukuman (DS7 "Pengingat berikutnya: konfirmasi akhir masa hukuman") belum dijadwalkan; perlu
   job terjadwal?
