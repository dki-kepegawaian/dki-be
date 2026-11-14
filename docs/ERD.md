# ERD Teralur: Fondasi + Cuti

Cakupan: fondasi (wilayah Sudin, sekolah, pendidik, akun, berkas, notifikasi, log aktivitas, nomor registrasi) dan modul Cuti (alur C1–C6 di Figma Teralur v0.9). Modul lain (Pensiun, Kenaikan Pangkat, dan seterusnya) akan menempel ke tabel fondasi yang sama.

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
    }

    sekolah {
        uuid id PK
        uuid sudin_id FK
        varchar npsn UK
        varchar nama
        varchar jenjang "SD, SMP, SMA, SMK, SLB"
        varchar kecamatan
        text alamat
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
