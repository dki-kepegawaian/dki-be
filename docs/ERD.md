# ERD Teralur

Bagian pertama: fondasi (wilayah Sudin, sekolah, pendidik, akun, berkas, notifikasi, log aktivitas, nomor registrasi) dan modul Cuti (alur C1–C6 di Figma Teralur v0.9). Bagian selanjutnya: satu modul per bagian, semuanya menempel ke tabel fondasi yang sama. Kolom baru di fondasi sejak versi awal: `users.username`, `users.nip`, `pendidik.tanggal_lahir`, `pendidik.tmt_cpns`, `pendidik.tmt_golongan` (syarat 4 tahun kenaikan pangkat), `sekolah.kepala_sekolah`, dan kontak layanan di `sudin`.

```mermaid
erDiagram
    sudin ||--o{ sekolah : "membawahi"
    sudin ||--o{ users : "staf / pimpinan"
    sekolah ||--o{ users : "operator"
    sekolah ||--o{ pendidik : "unit kerja"
    users ||--o{ refresh_tokens : ""
    users ||--o{ notifikasi : "penerima"
    users ||--o{ log_aktivitas : "pelaku"
    users ||--o{ dokumen : "pengunggah"
    sekolah ||--o{ dokumen : "pemilik akses"

    pendidik ||--o{ cuti : "mengajukan"
    sekolah ||--o{ cuti : ""
    users ||--o{ cuti : "diajukan_oleh"
    dokumen ||--o| cuti : "berkas_permohonan"
    dokumen ||--o| cuti : "surat_izin"
    cuti ||--o{ cuti_riwayat : "jejak status"
    users ||--o{ cuti_riwayat : "oleh"

    sudin {
        uuid id PK
        varchar kode UK "JB, JP, JS, JT, JU"
        varchar nama UK "Jakarta Barat"
        varchar kontak_unit "kontak layanan untuk sekolah"
        varchar kontak_telepon
        varchar kontak_email
        varchar jam_layanan
    }

    sekolah {
        uuid id PK
        uuid sudin_id FK
        varchar npsn UK
        varchar nama
        varchar jenjang "SD, SMP, SMA, SMK, SLB"
        varchar kecamatan
        text alamat
        varchar kepala_sekolah
    }

    users {
        uuid id PK
        varchar name
        varchar email UK
        varchar password
        varchar role "staf_sudin, operator_sekolah, pimpinan, admin"
        uuid sudin_id FK "wajib untuk staf_sudin dan pimpinan"
        uuid sekolah_id FK "wajib untuk operator_sekolah"
        varchar telp_number
        bool is_verified
        bool is_aktif "false = dinonaktifkan Sudin"
    }

    refresh_tokens {
        uuid id PK
        uuid user_id FK
        varchar token UK
        timestamptz expires_at
    }

    pendidik {
        uuid id PK
        uuid sekolah_id FK
        varchar nama
        varchar nip UK "null untuk non-PNS"
        varchar nik
        varchar nuptk
        varchar nrk
        varchar nikki
        varchar status_kepegawaian "PNS, PPPK, PPPK_PW, KKI"
        varchar pangkat "Penata"
        varchar golongan "III/c"
        varchar jabatan "sesuai SK"
        varchar jabatan_dapodik
        varchar program_studi
        varchar sertifikasi
        varchar email
        varchar telepon
        text alamat_domisili
        varchar kota_domisili
    }

    dokumen {
        uuid id PK
        uuid sekolah_id FK "dipakai untuk cek akses unduh"
        uuid diunggah_oleh FK
        varchar nama_asli
        varchar path "object key di bucket Garage: dokumen/2026/11/{id}.pdf"
        varchar mime_type
        bigint ukuran
    }

    notifikasi {
        uuid id PK
        uuid user_id FK
        varchar judul
        text pesan
        varchar tautan "/cuti/{id}"
        timestamptz dibaca_at "null = belum dibaca"
    }

    log_aktivitas {
        uuid id PK
        uuid user_id FK
        varchar aksi "cuti.ajukan, cuti.periksa, ..."
        varchar entitas "cuti"
        uuid entitas_id
        text keterangan
    }

    nomor_urut {
        varchar prefix PK "CUT, PEN, KP, ..."
        int tahun PK
        int terakhir
    }

    cuti {
        uuid id PK
        varchar nomor UK "CUT-2026-0142"
        uuid pendidik_id FK
        uuid sekolah_id FK
        uuid diajukan_oleh FK
        varchar jenis "tahunan, besar, sakit, melahirkan, alasan_penting, di_luar_tanggungan_negara"
        date tanggal_mulai
        date tanggal_selesai
        int hari_kerja
        text alasan
        bool disetujui_kepsek
        uuid berkas_permohonan_id FK
        varchar status "diajukan, perlu_perbaikan, menunggu_surat_fisik, diproses_sudin, disetujui, ditolak"
        bool berkas_lengkap "hasil periksa C3"
        bool berkas_terbaca "hasil periksa C3"
        text catatan_perbaikan
        uuid diperiksa_oleh FK
        timestamptz diperiksa_at
        date surat_fisik_diterima_tanggal "C4"
        uuid surat_fisik_penerima_id FK
        bool nota_dinas_ada
        text catatan_penerimaan
        varchar nomor_surat_izin "C5"
        date tanggal_surat_izin
        uuid surat_izin_id FK
        text catatan_keputusan
        uuid diputuskan_oleh FK
        timestamptz diputuskan_at
    }

    cuti_riwayat {
        uuid id PK
        uuid cuti_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

## Alur status Cuti

```mermaid
stateDiagram-v2
    [*] --> diajukan: Operator ajukan (C1)
    diajukan --> perlu_perbaikan: Sudin periksa, berkas tidak lengkap/terbaca (C3)
    perlu_perbaikan --> diajukan: Operator unggah ulang berkas (C6)
    diajukan --> menunggu_surat_fisik: Sudin periksa, berkas OK (C3)
    menunggu_surat_fisik --> diproses_sudin: Surat fisik + nota dinas diterima (C4)
    diproses_sudin --> disetujui: Catat keputusan (C5)
    diproses_sudin --> ditolak: Catat keputusan (C5)
    disetujui --> [*]
    ditolak --> [*]
```

## Catatan desain

- **Satu baris per pengajuan.** Langkah C3–C5 masing-masing terjadi sekali per pengajuan, jadi kolomnya disimpan langsung di `cuti`. Jejak lengkapnya (termasuk perbaikan berulang) ada di `cuti_riwayat`.
- **Scope akses.** Staf Sudin dan Pimpinan hanya melihat sekolah di Sudin-nya, Operator hanya melihat sekolahnya sendiri. `cuti.sekolah_id` disimpan ulang (denormalisasi) supaya filter antrean tidak perlu join ke `pendidik`.
- **Nomor registrasi.** `nomor_urut` dinaikkan secara atomik (`INSERT ... ON CONFLICT DO UPDATE ... RETURNING`), lalu dipakai bersama oleh modul lain dengan prefix berbeda.
- **Akun.** Tidak ada pendaftaran mandiri. Staf Sudin membuat akun operator sekolah dan sesama staf di wilayahnya; akun pimpinan dan admin dibuat admin. Akun tidak dihapus, hanya dinonaktifkan (`is_aktif = false`), supaya jejak `cuti_riwayat` dan `log_aktivitas` tetap utuh.
- **Berkas.** Isi file disimpan di object storage Garage (S3-compatible, bucket privat). Tabel `dokumen` hanya menyimpan metadata dan object key. Unduhan lewat `GET /api/dokumen/{id}` dengan cek akses berdasarkan `dokumen.sekolah_id`.
- **Sisa cuti tahunan.** Jatah 12 hari kerja per tahun dikurangi cuti tahunan yang belum ditolak. Hari kerja dihitung Senin–Jumat. Libur nasional, cuti bersama, dan akumulasi sisa tahun lalu belum dihitung.
- **Keputusan final.** Setelah `disetujui` atau `ditolak`, pengajuan tidak bisa diubah (sesuai peringatan di layar C5).

---

## Modul Pensiun

## Pensiun

```mermaid
erDiagram
    sekolah ||--o{ pensiun : "sekolah_id (denormalisasi untuk scope)"
    pendidik ||--o| pensiun : "satu pengajuan per pendidik"
    users ||--o{ pensiun : "diajukan_oleh / diverifikasi_oleh / dikirim_oleh / selesai_oleh"
    pensiun ||--|{ pensiun_berkas : "14 berkas"
    dokumen ||--o| pensiun_berkas : "dokumen_id (unik)"
    dokumen ||--o| pensiun : "bukti_kirim_id, sk_id"
    pensiun ||--|{ pensiun_riwayat : "riwayat status"
    users ||--o{ pensiun_riwayat : oleh

    pensiun {
        uuid id PK
        varchar nomor UK "PEN-YYYY-NNNN"
        uuid pendidik_id FK
        uuid sekolah_id FK
        uuid diajukan_oleh FK
        varchar jenis "batas_usia | atas_permintaan_sendiri"
        date tmt_pensiun
        varchar status
        uuid diverifikasi_oleh
        timestamptz diverifikasi_at
        varchar nomor_surat_pengantar
        date tanggal_kirim
        varchar cara_kirim
        uuid bukti_kirim_id FK
        text catatan_kirim
        uuid dikirim_oleh
        varchar nomor_sk
        date tanggal_sk
        uuid sk_id FK
        uuid selesai_oleh
        timestamptz selesai_at
    }
    pensiun_berkas {
        uuid id PK
        uuid pensiun_id FK
        varchar jenis "kode dari 14 berkas P2"
        uuid dokumen_id FK "unik"
        bool valid "null = belum dinilai"
        text catatan "alasan tidak valid"
        uuid dinilai_oleh
        timestamptz dinilai_at
    }
    pensiun_riwayat {
        uuid id PK
        uuid pensiun_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

```mermaid
stateDiagram-v2
    [*] --> diajukan: operator ajukan (14 berkas)
    diajukan --> perlu_perbaikan: Sudin verifikasi, ada berkas tidak valid
    perlu_perbaikan --> diajukan: operator unggah ulang berkas yang ditolak
    diajukan --> terverifikasi: Sudin verifikasi, semua valid
    terverifikasi --> diusulkan_ke_bkd: Sudin catat pengiriman ke BKD
    diusulkan_ke_bkd --> selesai: Sudin unggah SK
    selesai --> [*]
```

Catatan desain:
- Berkas diunggah satu per satu (`POST /api/pensiun/berkas`) lalu dirujuk lewat id saat mengajukan; 14 PDF × 2 MB tidak muat di batas body multipart 11 MiB.
- Penilaian disimpan per baris `pensiun_berkas`. Perbaikan mengganti `dokumen_id` pada baris yang tidak valid dan mengosongkan penilaiannya; dokumen lama tetap ada di tabel `dokumen`.
- `dokumen_id` unik per tabel berkas, jadi satu dokumen tidak bisa dipakai dua pengajuan pensiun.
- Struktur kolom penilaian (`BerkasPersyaratan`) dan logikanya (`pkg/persyaratan`) dipakai bersama dengan Kenaikan Pangkat; tabelnya terpisah.

---

## Modul Kenaikan Pangkat

## Kenaikan Pangkat

```mermaid
erDiagram
    sekolah ||--o{ kenaikan_pangkat : "sekolah_id (denormalisasi untuk scope)"
    pendidik ||--o{ kenaikan_pangkat : "maks. satu yang belum selesai; K5 memperbarui pangkat/golongan"
    users ||--o{ kenaikan_pangkat : "diajukan_oleh / diverifikasi_oleh / dikirim_oleh / selesai_oleh"
    kenaikan_pangkat ||--|{ kenaikan_pangkat_berkas : "6 berkas"
    dokumen ||--o| kenaikan_pangkat_berkas : "dokumen_id (unik)"
    dokumen ||--o{ kenaikan_pangkat : "bukti_kirim_id (satu per sekolah per batch), sk_id"
    kenaikan_pangkat ||--|{ kenaikan_pangkat_riwayat : "riwayat status"
    users ||--o{ kenaikan_pangkat_riwayat : oleh

    kenaikan_pangkat {
        uuid id PK
        varchar nomor UK "KP-YYYY-NNNN"
        uuid pendidik_id FK
        uuid sekolah_id FK
        uuid diajukan_oleh FK
        date periode "1 April / 1 Oktober"
        varchar pangkat_lama
        varchar golongan_lama
        varchar pangkat_baru
        varchar golongan_baru
        varchar status
        uuid diverifikasi_oleh
        timestamptz diverifikasi_at
        varchar nomor_surat_pengantar
        date tanggal_kirim
        varchar cara_kirim
        uuid bukti_kirim_id FK
        text catatan_kirim
        uuid dikirim_oleh
        varchar nomor_sk
        date tanggal_sk
        date tmt_pangkat
        uuid sk_id FK
        uuid selesai_oleh
        timestamptz selesai_at
    }
    kenaikan_pangkat_berkas {
        uuid id PK
        uuid kenaikan_pangkat_id FK
        varchar jenis "kode dari 6 berkas K2"
        uuid dokumen_id FK "unik"
        bool valid "null = belum dinilai"
        text catatan
        uuid dinilai_oleh
        timestamptz dinilai_at
    }
    kenaikan_pangkat_riwayat {
        uuid id PK
        uuid kenaikan_pangkat_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

```mermaid
stateDiagram-v2
    [*] --> diajukan: operator ajukan (6 berkas)
    diajukan --> perlu_perbaikan: Sudin verifikasi, ada berkas tidak valid
    perlu_perbaikan --> diajukan: operator unggah ulang berkas yang ditolak
    diajukan --> terverifikasi: Sudin verifikasi, semua valid
    terverifikasi --> diusulkan_ke_bkd: Sudin catat pengiriman (batch)
    diusulkan_ke_bkd --> selesai: Sudin unggah SK, pangkat pendidik diperbarui
    selesai --> [*]
```

Catatan desain:
- Pangkat/golongan lama dan baru disalin ke baris usulan saat mengajukan, supaya riwayat tetap benar setelah data induk berubah.
- K5 mengunci baris pendidik (`FOR UPDATE`) dan hanya memperbarui `pangkat`, `golongan`, `updated_at`; perubahan dicatat di `log_aktivitas` (`pendidik.ubah_pangkat`).
- Pengiriman BKD (K4) bersifat batch dan all-or-nothing seperti "terima surat fisik" di Cuti.
- Struktur kolom penilaian (`BerkasPersyaratan`) dan logikanya (`pkg/persyaratan`) dipakai bersama dengan Pensiun; tabelnya terpisah.

---

## Modul Sikap Disiplin

## Sikap Disiplin

Kasus dugaan pelanggaran disiplin (Figma DS1–DS9). Satu baris `disiplin` per kasus; semua tahap (verifikasi, pemeriksaan, rekomendasi, keputusan pimpinan, keputusan final, tindak lanjut) disimpan sebagai kolom di baris yang sama. Berkas ditautkan lewat `disiplin_berkas`, jejak status di `disiplin_riwayat`.

```mermaid
erDiagram
    pendidik ||--o{ disiplin : "terlapor"
    sekolah ||--o{ disiplin : "unit kerja (scope)"
    users ||--o{ disiplin : "dilaporkan_oleh / diverifikasi_oleh / direkomendasikan_oleh / diputuskan_oleh / ditutup_oleh"
    disiplin ||--o{ disiplin_berkas : "lampiran"
    dokumen ||--|| disiplin_berkas : ""
    disiplin ||--o{ disiplin_riwayat : "jejak status"
    users ||--o{ disiplin_riwayat : "oleh"

    disiplin {
        uuid id PK
        varchar nomor UK "DSP-2026-0007"
        uuid pendidik_id FK
        uuid sekolah_id FK "denormalisasi untuk akses.Scope"
        uuid dilaporkan_oleh FK
        varchar status
        date tenggat "tenggat tahap berjalan"
        varchar jenis_pelanggaran
        text uraian
        jsonb kronologi "[{tanggal, kejadian}]"
        bool cek_identitas "DS2 daftar periksa (4 kolom cek_*)"
        varchar keputusan_verifikasi "layak, perlu_kelengkapan, tidak_layak"
        text catatan_verifikasi "daftar kekurangan / alasan"
        date batas_kelengkapan
        text tanggapan_sekolah "DS3"
        varchar nomor_surat_panggilan "DS4, internal"
        timestamptz waktu_pemeriksaan "internal"
        text ringkasan_pemeriksaan "internal"
        text ringkasan_fakta "DS5, internal"
        varchar dugaan_terbukti "internal"
        varchar dasar_hukum "internal"
        varchar rekom_tingkat "internal"
        varchar rekom_jenis "internal"
        text justifikasi "internal"
        varchar keputusan_pimpinan "DS6: setujui, ubah, kembalikan"
        text catatan_pimpinan "internal"
        varchar tingkat "hukuman yang ditetapkan"
        varchar jenis_hukuman
        uuid diputuskan_oleh FK "pimpinan"
        varchar nomor_keputusan "DS7"
        date tanggal_keputusan
        date masa_mulai
        date masa_selesai "dihitung dari jenis hukuman"
        date tanggal_diterima_pegawai
        bool dicatat_idis "khusus PNS"
        varchar upaya_administratif "tidak_ada, ada"
        text catatan_upaya
        text catatan_pelaksanaan "DS8 dari sekolah"
        timestamptz ditutup_at
    }

    disiplin_berkas {
        uuid id PK
        uuid disiplin_id FK
        uuid dokumen_id FK, UK
        varchar jenis "bukti_awal, bap, salinan_keputusan, ..."
        bool internal "true = tidak untuk sekolah"
        timestamptz created_at
    }

    disiplin_riwayat {
        uuid id PK
        uuid disiplin_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

```mermaid
stateDiagram-v2
    [*] --> registrasi : operator lapor (DS1)
    registrasi --> perlu_kelengkapan : staf: perlu kelengkapan (DS2)
    perlu_kelengkapan --> registrasi : operator lengkapi (DS3)
    registrasi --> tidak_dilanjutkan : staf: tidak layak (DS2)
    registrasi --> pemeriksaan : staf: layak (DS2)
    pemeriksaan --> pemeriksaan : staf simpan pemanggilan/BAP (DS4)
    pemeriksaan --> menunggu_keputusan : staf ajukan rekomendasi (DS5)
    menunggu_keputusan --> pemeriksaan : pimpinan kembalikan untuk telaah (DS6)
    menunggu_keputusan --> sanksi_ditetapkan : pimpinan setujui / ubah tingkat (DS6)
    sanksi_ditetapkan --> tindak_lanjut : staf catat keputusan final (DS7)
    tindak_lanjut --> tindak_lanjut : staf tindak lanjut, operator laporan pelaksanaan (DS7, DS8)
    tindak_lanjut --> selesai : staf tutup kasus (DS7)
    selesai --> [*]
    tidak_dilanjutkan --> [*]
```

Catatan desain:

- Akses: semua query lewat `akses.Scope(actor, "disiplin.sekolah_id")`; di luar scope = 404. Penetapan keputusan (DS6) hanya `pimpinan` (route), staf dan admin mendapat 403.
- Kerahasiaan: kolom bertanda "internal" dan berkas `internal=true` (BAP, tanggapan pegawai, bukti tambahan, bukti penyampaian) tidak dikirim ke `operator_sekolah`. Dokumen internal disimpan dengan `dokumen.sekolah_id = NULL` sehingga tidak bisa diunduh sekolah lewat `/api/dokumen/{id}`; semua berkas kasus diunduh lewat `/api/disiplin/{id}/dokumen/{dokumen_id}`.
- Kolom tahap disimpan rata di satu tabel (tanpa tabel per tahap) karena tiap tahap hanya terjadi sekali per kasus; pengembalian oleh pimpinan menimpa rekomendasi lama, jejaknya tetap di `disiplin_riwayat`.

---

## Modul Generator Surat

## Generator Surat

```mermaid
erDiagram
    sudin ||--o{ sekolah : membawahi
    sekolah ||--o{ pendidik : menempatkan
    sekolah ||--o{ surat : "sekolah_id (scope)"
    pendidik ||--o{ surat : untuk
    users ||--o{ surat : "dibuat_oleh, final_oleh, diserahkan_oleh"
    dokumen |o--o| surat : "dokumen_id (PDF sistem)"
    dokumen |o--o| surat : "dokumen_ttd_id (hasil pindai)"
    surat ||--o{ surat_riwayat : mencatat
    users ||--o{ surat_riwayat : oleh

    surat {
        uuid id PK
        varchar nomor UK "0188/PK.01.02/2027"
        varchar jenis "tunjangan, aktif_mengajar, ..."
        uuid pendidik_id FK
        uuid sekolah_id FK "denormalisasi untuk akses.Scope"
        uuid dibuat_oleh FK
        varchar perihal
        date tanggal_surat
        text keterangan
        varchar status "draf, menunggu_ttd, final, dibatalkan"
        uuid dokumen_id FK "nullable"
        uuid dokumen_ttd_id FK "nullable"
        uuid final_oleh FK "nullable"
        timestamptz final_at
        uuid diserahkan_oleh FK "nullable"
        timestamptz diserahkan_at
        text alasan_batal
        timestamptz created_at
        timestamptz updated_at
    }
    surat_riwayat {
        uuid id PK
        uuid surat_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

### Alur status Generator Surat

```mermaid
stateDiagram-v2
    [*] --> draf: simpan draf (S2)
    draf --> draf: ubah isian / pratinjau (S3)
    draf --> menunggu_ttd: buat dokumen PDF (S3)
    menunggu_ttd --> final: unggah hasil pindai bertanda tangan (S4)
    draf --> dibatalkan: batalkan
    menunggu_ttd --> dibatalkan: batalkan
    final --> [*]: tandai diserahkan (S5, status tetap final)
    dibatalkan --> [*]
```

Catatan desain:

- Template surat (judul, pembuka, penutup, awalan nama berkas) ada di `modules/surat/service/template.go`, satu entri per jenis. Isian `keterangan` dari form disisipkan apa adanya; data guru (nama, NIP, pangkat/golongan, jabatan, unit kerja) diambil dari master pendidik saat PDF dirender.
- Kop dan penanda tangan diturunkan dari Sudin sekolah pendidik, tidak disimpan di tabel.
- Nomor surat urut per tahun tanggal surat lewat `nomor_urut` (prefix `SRT`), format `NNNN/PK.01.02/YYYY`. Nomor isian manual boleh, unik global.
- PDF sistem dan hasil pindai disimpan di tabel `dokumen` dengan `sekolah_id` pendidik, jadi ikut aturan akses unduh dokumen yang sudah ada.

---

## Modul Analisis Jabatan

## Analisis Jabatan (Anjab)

```mermaid
erDiagram
    sudin ||--o{ anjab_periode : "memiliki"
    anjab_periode ||--o{ anjab_beban : "beban JP per mapel per tingkat"
    anjab_periode ||--o{ anjab_sekolah : "peserta"
    sekolah ||--o{ anjab_sekolah : "data per periode"
    anjab_sekolah ||--o{ anjab_rombel : "rombel per tingkat"
    anjab_sekolah ||--o{ anjab_guru : "guru dihitung"
    pendidik ||--o{ anjab_guru : "guru"
    anjab_sekolah ||--o{ anjab_riwayat : "riwayat status"
    users ||--o{ anjab_riwayat : "oleh"

    anjab_periode {
        uuid id PK
        uuid sudin_id FK "unik bersama tahun_ajaran"
        varchar tahun_ajaran "2026/2027"
        varchar jenjang "SD,SMP,SMA,SMK"
        date tanggal_buka
        date tanggal_tutup "tenggat, bisa diperpanjang"
        varchar acuan_peraturan
        int jam_wajib "JP per minggu"
        int rasio_sd
        int rasio_smp
        int rasio_sma "SMA dan SMK"
        date batas_tmt_pensiun
        uuid dibuat_oleh FK
        uuid dibuka_oleh FK
        timestamptz dibuka_at "terisi = parameter terkunci"
    }
    anjab_beban {
        uuid id PK
        uuid periode_id FK
        varchar jenjang
        varchar mapel
        int tingkat "1-12"
        int jp
    }
    anjab_sekolah {
        uuid id PK
        uuid periode_id FK "unik bersama sekolah_id"
        uuid sekolah_id FK "scope akses"
        varchar status "draf|dikirim|dikembalikan|ditetapkan"
        int kiriman_ke
        uuid dikirim_oleh FK
        timestamptz dikirim_at
        text catatan_sudin
        uuid ditinjau_oleh FK
        timestamptz ditinjau_at
        text hasil "JSON hasil hitungan, dibekukan saat ditetapkan"
    }
    anjab_rombel {
        uuid id PK
        uuid anjab_sekolah_id FK "unik bersama tingkat"
        uuid sekolah_id FK
        int tingkat
        int jumlah_rombel
        int jumlah_siswa
    }
    anjab_guru {
        uuid id PK
        uuid anjab_sekolah_id FK "unik bersama pendidik_id"
        uuid sekolah_id FK
        uuid pendidik_id FK
        varchar mapel_sk
        varchar mapel_riil
        int jp_riil
    }
    anjab_riwayat {
        uuid id PK
        uuid anjab_sekolah_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

```mermaid
stateDiagram-v2
    [*] --> draf: impor master / buka masa pembaruan
    draf --> dikirim: operator kirim
    dikirim --> dikembalikan: Sudin kembalikan (catatan wajib)
    dikembalikan --> dikirim: operator perbaiki & kirim ulang
    dikirim --> ditetapkan: Sudin tetapkan
    ditetapkan --> [*]
```

Catatan desain:

- Periode milik satu Sudin (`anjab_periode.sudin_id` diambil dari akun staf, bukan dari body). Akses periode: staf/pimpinan
  Sudin itu, operator sekolah di Sudin itu, admin semua. Data per sekolah (`anjab_sekolah`, `anjab_rombel`, `anjab_guru`)
  memakai `sekolah_id` dan `akses.Scope`; di luar scope = 404.
- Parameter dan beban JP terkunci setelah `dibuka_at` terisi. Impor master hanya sebelum masa pembaruan dibuka.
- Hitungan adalah fungsi murni di `modules/anjab/hitung` (dua basis: mapel SK dan mapel riil). Sebelum ditetapkan, hasil
  dihitung ulang dari data terkini; saat ditetapkan hasilnya disimpan sebagai JSON di `anjab_sekolah.hasil` sehingga
  perubahan pendidik (mis. tanggal lahir) atau mutasi kemudian tidak mengubah hasil yang sudah ditetapkan.
- Guru hasil impor di-upsert ke tabel `pendidik` (berdasar NIP; tanpa NIP berdasar nama di sekolah yang sama).
- Monitoring (A6) hanya memakai sekolah yang sudah ditetapkan, versi kondisi riil.

---

## Modul Layanan Dapodik

## Layanan Dapodik

```mermaid
erDiagram
    sekolah ||--o{ dapodik_tiket : "pemilik tiket"
    users ||--o{ dapodik_tiket : "diajukan_oleh / diselesaikan_oleh"
    pendidik |o--o{ dapodik_tiket : "PTK terkait"
    dapodik_tiket ||--o{ dapodik_lampiran : ""
    dokumen ||--o{ dapodik_lampiran : "berkas"
    dapodik_tiket ||--o{ dapodik_riwayat : "jejak status"
    users ||--o{ dapodik_riwayat : "oleh"

    dapodik_tiket {
        uuid id PK
        varchar nomor UK "DPK-2026-0192"
        uuid sekolah_id FK
        uuid diajukan_oleh FK
        varchar jenis "revisi_akun_operator, autentikasi_ptk, kode_referral_yayasan, perbaikan_data_ptk, lainnya"
        uuid pendidik_id FK "nullable"
        jsonb isian "field formulir per jenis"
        text uraian
        varchar status "baru, perlu_info_tambahan, selesai"
        date tenggat "3 hari kerja sejak (re)submit"
        text permintaan_info
        text balasan_info
        text catatan_penyelesaian
        uuid diselesaikan_oleh FK
        timestamptz diselesaikan_at
        timestamptz created_at
        timestamptz updated_at
    }

    dapodik_lampiran {
        uuid id PK
        uuid tiket_id FK
        uuid dokumen_id FK
        varchar tahap "pengajuan, info_tambahan"
        timestamptz created_at
    }

    dapodik_riwayat {
        uuid id PK
        uuid tiket_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

### Alur status Layanan Dapodik

```mermaid
stateDiagram-v2
    [*] --> baru: Operator kirim tiket (DP1–DP2)
    baru --> perlu_info_tambahan: Sudin minta info tambahan (DP4)
    perlu_info_tambahan --> baru: Operator lengkapi info (DP5)
    baru --> selesai: Sudin proses di aplikasi Dapodik (di luar sistem) + catatan (DP4)
    selesai --> [*]
```

### Catatan desain

- **Formulir per jenis di `isian` (jsonb).** Lima jenis layanan punya field berbeda dan hanya DP2 (revisi akun operator) yang dirinci di Figma. Satu kolom jsonb `map[string]string` lebih sederhana daripada tabel per jenis; field wajib divalidasi di service.
- **Lampiran banyak.** Tiket bisa punya beberapa berkas (SK penunjukan + surat permohonan, lalu dokumen pendukung saat melengkapi), jadi dipisah ke `dapodik_lampiran` dengan `tahap`.
- **Scope.** `dapodik_tiket.sekolah_id` diambil dari penempatan operator (bukan dari body); staf dan pimpinan dibatasi Sudin lewat `akses.Scope`.
- **Tenggat.** Kolom `tenggat` (date) diisi 3 hari kerja setelah tiket masuk atau dilengkapi ulang, dipakai untuk urutan antrean DP3 dan penanda `terlambat`.
- **Final.** Status `selesai` tidak bisa diubah lagi.

---

## Modul Kinerja P3K Paruh Waktu

## Kinerja P3K Paruh Waktu

```mermaid
erDiagram
    pendidik ||--o{ kinerja_penilaian : "dinilai (PPPK_PW)"
    sekolah ||--o{ kinerja_penilaian : ""
    users ||--o{ kinerja_penilaian : "dinilai_oleh / difinalkan_oleh"
    dokumen ||--o| kinerja_penilaian : "dokumen_sistem (PDF)"
    dokumen ||--o| kinerja_penilaian : "dokumen_ttd (pindai)"
    kinerja_penilaian ||--o{ kinerja_riwayat : "jejak status"
    users ||--o{ kinerja_riwayat : "oleh"

    kinerja_penilaian {
        uuid id PK
        varchar nomor UK "P3K-2026-0112"
        uuid pendidik_id FK "UK bersama tahun+semester"
        int tahun
        int semester "1 = Jan–Jun, 2 = Jul–Des"
        uuid sekolah_id FK
        uuid dinilai_oleh FK
        varchar status "draf, menunggu_ttd, final"
        int berorientasi_pelayanan "1–5, null = belum"
        int akuntabel
        int kompeten
        int harmonis
        int loyal
        int adaptif
        int kolaboratif
        numeric nilai_akhir "rata-rata, 1 desimal"
        varchar predikat
        uuid dokumen_sistem_id FK
        timestamptz dokumen_dibuat_at
        uuid dokumen_ttd_id FK
        uuid difinalkan_oleh FK
        timestamptz difinalkan_at
        timestamptz created_at
        timestamptz updated_at
    }

    kinerja_riwayat {
        uuid id PK
        uuid penilaian_id FK
        varchar dari_status
        varchar ke_status
        uuid oleh FK
        text catatan
        timestamptz created_at
    }
```

### Alur status Kinerja P3K PW

```mermaid
stateDiagram-v2
    [*] --> draf: Operator isi nilai BerAKHLAK (KP1–KP2)
    draf --> draf: Simpan draf (ubah nilai)
    draf --> menunggu_ttd: Buat dokumen penilaian PDF, 7 parameter terisi (KP2–KP3)
    menunggu_ttd --> final: Staf Sudin unggah hasil pindai bertanda tangan (KP4)
    final --> [*]
```

### Catatan desain

- **Satu penilaian per pegawai per periode.** Unique index `(pendidik_id, tahun, semester)`; pembuatan juga mengunci baris pendidik lalu mengecek dulu supaya pesan 409 jelas.
- **Tujuh kolom nilai.** Parameter BerAKHLAK tetap (dari KP2), jadi disimpan sebagai kolom `int` nullable, bukan tabel detail. `nilai_akhir` dan `predikat` dihitung ulang setiap simpan.
- **Dua dokumen.** `dokumen_sistem_id` = PDF yang dibuat server (go-pdf/fpdf, disimpan lewat `dokumen.SimpanBytes`); `dokumen_ttd_id` = hasil pindai bertanda tangan yang diunggah staf Sudin. Keduanya ber-`sekolah_id` sekolah pegawai sehingga operator sekolah itu dan staf Sudin-nya bisa mengunduh.
- **Final.** Status `final` tidak bisa diubah; nilai hanya bisa diubah saat `draf`.

---

## Modul Arsip Kepegawaian

```mermaid
erDiagram
    pendidik ||--o{ arsip : "memiliki"
    sekolah ||--o{ arsip : "scope akses"
    arsip ||--o{ arsip_versi : "riwayat versi"
    dokumen ||--o| arsip_versi : "berkas utama / lampiran"
    arsip ||--o{ arsip_log : "aktivitas"
    users ||--o{ arsip_log : "pelaku"

    arsip {
        uuid id PK
        uuid pendidik_id FK
        uuid sekolah_id FK
        varchar kategori "sk, tunjangan, ijazah, kinerja, surat, lainnya"
        varchar jenis_dokumen
        varchar nomor
        date tanggal_dokumen
        int versi_terkini
        varchar sumber_modul "kosong = unggahan manual"
        uuid sumber_id
    }
    arsip_versi {
        uuid id PK
        uuid arsip_id FK
        int versi "unik per arsip"
        uuid dokumen_id FK
        uuid lampiran_dokumen_id FK
        text catatan
        uuid diunggah_oleh FK
    }
    arsip_log {
        uuid id PK
        uuid arsip_id FK
        int versi
        varchar aksi "melihat, mengunduh, mengunggah, mengganti_versi"
        uuid user_id FK
    }
```

Dokumen final dari modul lain masuk lewat `pkg/arsipkan.Otomatis` di transaksi yang sama dengan langkah final modul tersebut. Kasus disiplin tidak diarsipkan di sini karena bersifat terbatas.

---

## Modul Dasbor Lintas Modul

Dasbor tidak punya tabel sendiri. Ia membaca tabel modul (cuti, pensiun, kenaikan_pangkat, disiplin, dapodik_tiket, kinerja_penilaian, surat, anjab_sekolah) dengan scope peran yang sama. Spesifikasi status dan tugas per modul ada di `modules/dasbor/spek`.
