# Layanan Dapodik: catatan implementasi

## Endpoint (`/api/dapodik`)

| Metode | Path | Peran | Fungsi |
|---|---|---|---|
| GET | `/api/dapodik` | semua (ter-scope) | Daftar tiket; filter `status`, `jenis`, `sekolah_id`, `search`, `urut=tenggat` |
| GET | `/api/dapodik/ringkasan` | semua | Jumlah tiket per status |
| GET | `/api/dapodik/{id}` | semua | Detail + lampiran + riwayat |
| POST | `/api/dapodik` | operator_sekolah | Kirim tiket (multipart: `jenis`, `uraian`, `isian` JSON, `pendidik_id`, `lampiran[]`) |
| POST | `/api/dapodik/{id}/lengkapi` | operator_sekolah | `perlu_info_tambahan` → `baru` (multipart: `balasan`, `lampiran[]`) |
| POST | `/api/dapodik/{id}/tinjau` | staf_sudin | `baru` → `perlu_info_tambahan` atau `selesai` (JSON: `keputusan`, `catatan`) |

Pimpinan dan admin hanya membaca. Data di luar scope menghasilkan 404.

## Asumsi

- Nomor tiket memakai prefix `DPK` (layar DP1–DP5). Layar beranda/notifikasi sekolah memakai `DPD-2026-0057`; dianggap salah ketik di data contoh.
- "Admin Dapodik Sudin" = peran `staf_sudin` di Sudin sekolah tersebut. Tidak ada peran terpisah.
- Field formulir per jenis disimpan di `isian` (jsonb). Hanya revisi akun operator (DP2) dan perbaikan data PTK (DP5) yang dirinci di Figma, jadi hanya keduanya yang punya field wajib:
  - `revisi_akun_operator`: `nama_operator_lama`, `nama_operator_baru`, `nik_operator_baru` (16 digit), `email_operator_baru`, plus minimal 2 lampiran (SK penunjukan operator dan surat permohonan). Lampiran tidak diberi label; urutan unggah mengikuti formulir.
  - `perbaikan_data_ptk`: `pendidik_id` (PTK di sekolah operator) dan `data_diperbaiki`.
  - Jenis lain cukup `uraian`.
- Batas berkas mengikuti layanan dokumen bersama (PDF/JPG/PNG, 10 MB), bukan 2 MB seperti teks di DP2.
- Tenggat tinjau = 3 hari kerja (Senin–Jumat) sejak tiket masuk; di-reset saat operator melengkapi info. Contoh tanggal di DP3 tidak konsisten (ada yang 3 hari kalender), jadi diambil hari kerja. Libur nasional belum dihitung.
- Dokumen pendukung saat melengkapi (DP5) opsional di API karena permintaan info bisa berupa teks saja; UI boleh mewajibkannya bila admin meminta dokumen.
- DP4 memperlihatkan NIK operator baru dalam bentuk tersamar. API mengembalikan NIK utuh ke operator sekolah pengaju dan staf Sudin (keduanya butuh untuk proses); penyamaran dilakukan di UI.
- `permintaan_info` dan `balasan_info` menyimpan yang terakhir; seluruh bolak-balik ada di `riwayat[].catatan`.

## Pertanyaan terbuka

1. Apakah tiket bisa dibatalkan operator atau ditolak Sudin (mis. permintaan tidak valid)? Kamus status hanya Baru, Perlu Info Tambahan, Selesai.
2. Field formulir untuk autentikasi PTK, kode referral yayasan, dan lainnya belum ada di Figma. Perlu daftar field wajibnya.
3. Berapa tenggat resmi tinjau tiket (hari kerja atau kalender)? Apakah tenggat dihitung ulang setelah operator melengkapi info?
4. "Unduh rekap" (DP3) memakai `GET /api/dasbor/pengajuan/ekspor?modul=dapodik` (CSV). Perlukah juga PDF/Excel?
