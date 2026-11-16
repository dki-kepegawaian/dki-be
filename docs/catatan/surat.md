# Generator Surat: catatan implementasi

Paket `modules/surat`, route `/api/surat`, tabel `surat` + `surat_riwayat` (migrasi `20261115040000_create_surat_tables`).

## Endpoint

| Metode | Path | Peran | Layar |
|---|---|---|---|
| GET | `/api/surat/jenis` | staf_sudin, admin, pimpinan | S1 kartu jenis surat |
| GET | `/api/surat` | staf_sudin, admin, pimpinan | S1 surat terakhir, S4 antrean `status=menunggu_ttd` |
| GET | `/api/surat/:id` | staf_sudin, admin, pimpinan | detail + riwayat |
| GET | `/api/surat/:id/pratinjau` | staf_sudin, admin, pimpinan | S3 pratinjau (PDF langsung) |
| POST | `/api/surat` | staf_sudin, admin | S2 simpan draf |
| PUT | `/api/surat/:id` | staf_sudin, admin | S3 ubah isian (hanya `draf`) |
| POST | `/api/surat/:id/buat-dokumen` | staf_sudin, admin | S3 simpan & buat dokumen: `draf` → `menunggu_ttd` |
| POST | `/api/surat/:id/final` | staf_sudin, admin | S4 unggah pindaian: `menunggu_ttd` → `final` |
| POST | `/api/surat/:id/batal` | staf_sudin, admin | `draf`/`menunggu_ttd` → `dibatalkan` |
| POST | `/api/surat/:id/serahkan` | staf_sudin, admin | S5 tandai sudah diserahkan (status tetap `final`) |

Operator sekolah mendapat 403 di semua endpoint. Surat di luar wilayah Sudin: 404.
PDF sistem dan hasil pindai diunduh lewat `/api/dokumen/:id` (URL ada di respons `dokumen.url` / `dokumen_ttd.url`).

## Asumsi

- **Pratinjau** dirender langsung dari isian dan data master saat itu, tanpa disimpan (paling sederhana; tidak ada berkas yatim). PDF baru disimpan sekali saat "Simpan & buat dokumen" (S3), dan itulah PDF yang dicetak. Frontend menyimpan draf dulu sebelum menekan "Pratinjau surat" di S2.
- **Nomor surat** `NNNN/PK.01.02/YYYY` mengikuti contoh Figma. Urutan per tahun tanggal surat lewat `helpers.NomorRegistrasi(tx, "SRT", tahun)`; kode klasifikasi `PK.01.02` dipakai untuk semua jenis karena Figma S1/S4 memakai kode itu untuk jenis yang berbeda-beda. Nomor bisa diisi manual (S2 "Nomor dicek unik saat disimpan"), unik global, 409 bila dipakai. Nomor diberikan saat draf disimpan; nomor surat yang dibatalkan tidak dipakai ulang. Nomor otomatis melompati nomor yang sudah dipakai isian manual.
- Tidak ada nomor registrasi terpisah (mis. `TUN-2026-0071`); teks Figma yang tersedia tidak memuatnya, dan breadcrumb S4/S5 memakai nomor surat sebagai identitas.
- **Kop dan penanda tangan** diturunkan dari nama Sudin sekolah pendidik ("Suku Dinas Pendidikan Kota Administrasi Jakarta Barat"). Data fondasi tidak punya nomor wilayah ("Wilayah I"), alamat, atau telepon kantor Sudin, jadi baris alamat kop di S3 tidak dicetak. Nama dan NIP pejabat tidak dicetak; ditulis tangan saat TTD basah, sesuai placeholder "(nama & NIP pejabat penanda tangan)".
- **Template** per jenis hanya beda judul, kalimat pembuka, dan penutup. Isi pokok (mis. jam mengajar, rincian tunjangan, tanggal cuti) diketik staf di `keterangan`. Placeholder yang didukung: `{penanda_tangan}`, `{nama}`, `{unit_kerja}`, `{perihal}`.
- Surat izin cuti / pengantar KP / pengantar pensiun **tidak ditautkan** ke pengajuan cuti, KP, atau pensiun; isian bebas.
- Data guru di PDF selalu dari master saat dirender. PDF yang sudah dibuat menjadi arsip tetap; pratinjau surat final bisa berbeda bila data master berubah.
- **Hasil pindai** wajib PDF dan maksimal 5 MB (teks S4), lebih ketat dari batas umum dokumen 10 MB.
- **Batal** wajib alasan (dicatat di riwayat dan `alasan_batal`). PDF yang sudah dibuat tidak dihapus.
- **Serahkan** tidak mengubah status (kamus status hanya empat), hanya mengisi `diserahkan_at`/`diserahkan_oleh` dan log aktivitas; hanya sekali.
- **Notifikasi**: buat dokumen → staf Sudin wilayah sekolah ("Surat … menunggu tanda tangan basah Kepala Sudin…", sesuai N1); final → operator sekolah pendidik, tautan ke `/api/dokumen/<id hasil pindai>` karena operator tidak punya akses ke `/surat`. Batal tidak mengirim notifikasi.
- Admin boleh mengelola surat untuk pendidik di sekolah mana pun.

## Pertanyaan terbuka

1. Apakah kode klasifikasi berbeda per jenis surat (arsip H4 memuat `PK.01.04` untuk surat keterangan masa kerja, `TU.03.01` untuk surat tugas)? Bila ya, cukup tambah kolom kode di template.
2. Apakah nomor urut dipakai bersama dengan surat lain Sudin (di luar Teralur) sehingga perlu diisi manual dari agenda surat?
3. Perlukah alamat, telepon, dan nomor wilayah kop surat disimpan di master Sudin?
4. Apakah Pimpinan (Kepala Sudin) perlu notifikasi saat ada surat menunggu tanda tangan?
5. Surat izin cuti: perlukah dibuat otomatis dari cuti yang disetujui, atau tetap manual seperti sekarang?
6. Apakah "Tandai sudah diserahkan" perlu mencatat penerima dan tanggal serah?
