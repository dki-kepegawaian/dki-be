# Teralur Backend (dki-be)

Backend layanan kepegawaian guru untuk Suku Dinas (Sudin) Pendidikan DKI Jakarta. Isinya:

- **Fondasi:** akun dan peran, data sekolah dan pendidik, dokumen, notifikasi.
- **Layanan guru:** cuti, pensiun, kenaikan pangkat, sikap disiplin, layanan Dapodik, penilaian kinerja P3K Paruh Waktu.
- **Pekerjaan Sudin:** generator surat, analisis jabatan, arsip kepegawaian.
- **Tampilan gabungan:** tugas, daftar pengajuan, dan rekap lintas modul.

Dibangun dengan Go (Gin, GORM), PostgreSQL, Garage (penyimpanan berkas yang kompatibel dengan S3), dan nginx. Semuanya dijalankan lewat Docker.

## Yang perlu dipasang

- **Docker**: Docker Desktop (Windows/macOS) atau Docker Engine dengan Compose v2 (Linux). Pastikan Docker sudah berjalan sebelum lanjut.
- **Git**

Go tidak perlu dipasang untuk menjalankan aplikasi lewat Docker.

## Mulai cepat

**1. Ambil kode**

```bash
git clone https://github.com/dki-kepegawaian/dki-be.git
cd dki-be
```

**2. Jalankan**

```bash
docker compose up -d --build
```

Pertama kali butuh beberapa menit karena Docker mengunduh image dan membangun aplikasi. Setelah itu hanya beberapa detik. Tabel database dan data awal dibuat otomatis, jadi tidak perlu membuat file `.env` atau menjalankan perintah lain.

**3. Cek hasilnya**

```bash
docker compose ps
```

Empat layanan (`app`, `postgres`, `garage`, `nginx`) harus berstatus `Up`. Lalu buka dokumentasi API di <http://localhost:81/docs>.

## Login pertama

Akun berikut dibuat otomatis. Semua akun contoh memakai password `teralur123`, kecuali admin.

| Peran | Nama pengguna | Email | Password |
|---|---|---|---|
| admin | `admin` | `admin@teralur.test` | `admin12345` |
| staf Sudin (Jakarta Barat) | `staf.jb` | `staf.jb@teralur.test` | `teralur123` |
| pimpinan (Jakarta Barat) | `pimpinan.jb` | `pimpinan.jb@teralur.test` | `teralur123` |
| operator SMP Negeri 45 | `operator.smpn45` | `operator.smpn45@teralur.test` | `teralur123` |
| operator SD Negeri Rawa Buaya 09 | `operator.sdnrawabuaya09` | `operator.sdnrawabuaya09@teralur.test` | `teralur123` |
| staf Sudin (Jakarta Selatan) | `staf.js` | `staf.js@teralur.test` | `teralur123` |

Masuk bisa memakai nama pengguna atau email.

Akun contoh hanya ada selama `SEED_DEMO=true` (bawaan). Tidak ada pendaftaran mandiri: akun dibuat oleh staf Sudin atau admin lewat `/api/akun`.

**Mencoba API dari halaman `/docs`:**

1. Buka `POST /api/auth/login`, klik **Test Request**, isi `username` (atau `email`) dan `password`, lalu kirim.
2. Salin `access_token` dari respons.
3. Di panel **Authentication** (Bearer Token), tempel token itu. Sekarang endpoint lain bisa dicoba.

Access token berlaku 15 menit. Setelah itu ambil token baru lewat `POST /api/auth/refresh` dengan `refresh_token` dari respons login.

Alur cuti: operator sekolah mengajukan, staf Sudin memeriksa berkas, staf menandai surat fisik diterima, lalu staf mencatat keputusan. Penjelasan tiap langkah ada di tag **Cuti** pada halaman `/docs`.

## Perintah sehari-hari

| Perintah | Fungsi |
|---|---|
| `docker compose ps` | Lihat status layanan |
| `docker compose logs -f app` | Lihat log aplikasi (keluar dengan Ctrl+C) |
| `docker compose up -d --build` | Terapkan perubahan kode (membangun ulang aplikasi) |
| `docker compose restart app` | Mulai ulang aplikasi saja |
| `docker compose down` | Hentikan semua layanan. Data tetap tersimpan |
| `docker compose down -v` | Hentikan dan **hapus semua data** (database dan berkas), kembali ke kondisi awal |
| `docker compose exec app /app/teralur --migrate:status` | Lihat migrasi yang sudah berjalan |
| `docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'` | Masuk ke database |

Pengguna Git Bash di Windows: awali perintah `exec` dengan `MSYS_NO_PATHCONV=1`, kalau tidak, Git Bash mengubah `/app/...` menjadi path Windows. PowerShell dan CMD tidak punya masalah ini.

Data disimpan di volume Docker (`postgres-data`, `garage-meta`, `garage-data`) dan bertahan walau layanan dihentikan.

## Layanan dan port

| Layanan | Fungsi | Alamat dari komputer Anda |
|---|---|---|
| nginx | Pintu masuk, batas unggahan 12 MB | <http://localhost:81> (bisa diakses dari jaringan) |
| app | API Go | <http://127.0.0.1:8888> (hanya komputer ini) |
| postgres | Database | `127.0.0.1:5434` (hanya komputer ini) |
| garage | Penyimpanan berkas (S3) | `127.0.0.1:3900` (hanya komputer ini) |

Postgres memakai port 5434 supaya tidak bentrok dengan Postgres lain di port 5432.

## Mengubah konfigurasi

Tanpa file `.env`, Docker Compose memakai nilai bawaan untuk pengembangan. Untuk mengubahnya, salin contohnya, edit, lalu jalankan ulang `docker compose up -d`:

```bash
cp .env.example .env              # PowerShell: Copy-Item .env.example .env
```

| Variabel | Fungsi | Bawaan |
|---|---|---|
| `NGINX_PORT` | Port nginx di komputer Anda | `81` |
| `GOLANG_PORT` | Port langsung ke aplikasi | `8888` |
| `DB_PORT` | Port Postgres di komputer Anda | `5434` |
| `S3_HOST_PORT` | Port Garage di komputer Anda (tidak ada di `.env.example`, tambahkan sendiri bila perlu) | `3900` |
| `DB_USER`, `DB_PASS`, `DB_NAME` | Akses database | `teralur` |
| `JWT_SECRET` | Kunci penandatangan token. Wajib, minimal 16 karakter | nilai contoh |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | Admin awal | `admin@teralur.test`, `admin12345` |
| `SEED_DEMO` | `true` membuat sekolah, pendidik, dan akun contoh | `true` |
| `PENGINGAT_TENGGAT` | `false` mematikan notifikasi "tenggat terlewat" yang dikirim tiap pagi pukul 08.00 WIB | `true` |
| `APP_ENV` | `production` mematikan halaman `/docs` | `localhost` |
| `DOCS_ENABLED` | `true` menyalakan `/docs` walau `APP_ENV=production` | `false` |
| `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `GARAGE_RPC_SECRET`, `GARAGE_ADMIN_TOKEN` | Rahasia Garage | nilai contoh |

**Penting:** sebagian nilai hanya dibaca saat pertama kali dibuat.

- Postgres hanya membaca `DB_USER`, `DB_PASS`, dan `DB_NAME` saat volume database dibuat.
- Admin hanya dibuat satu kali, jadi mengubah `ADMIN_EMAIL` atau `ADMIN_PASSWORD` sesudahnya tidak mengubah admin yang sudah ada.
- Perlakukan kunci S3 dan rahasia Garage dengan cara yang sama.

Kalau mengubah nilai-nilai itu setelah stack pernah dijalankan, jalankan `docker compose down -v` dulu (menghapus data), baru `docker compose up -d --build`.

## Untuk server sungguhan

Nilai bawaan (password admin, password database, kunci S3, `JWT_SECRET`) tertulis di repo ini dan hanya untuk pengembangan. Sebelum dipakai di server:

1. Salin `.env.example` ke `.env`, lalu ganti semua rahasia:
   - `JWT_SECRET` dan `GARAGE_RPC_SECRET`: `openssl rand -hex 32`
   - `S3_ACCESS_KEY`: `GK` diikuti 24 karakter hex (`echo GK$(openssl rand -hex 12)`)
   - `S3_SECRET_KEY`: `openssl rand -hex 32`
   - `ADMIN_PASSWORD`, `DB_PASS`, `GARAGE_ADMIN_TOKEN`: password acak yang panjang
2. Set `SEED_DEMO=false` dan `APP_ENV=production`.
3. Jalankan pertama kali dengan volume yang masih kosong. Volume yang pernah dibuat dengan nilai bawaan tetap menyimpan password lama dan akun contoh. Kalau ada, hapus dengan `docker compose down -v` (menghapus data).
4. Pasang HTTPS di depan nginx. nginx di repo ini hanya melayani HTTP.
5. Cukup buka satu port ke luar, yaitu port nginx. Port 8888, 5434, dan 3900 sudah dibatasi ke `127.0.0.1`. Tetap atur firewall server.
6. Cadangkan volume `postgres-data`, `garage-meta`, dan `garage-data`.

## Mengembangkan kode

**Semua di Docker.** Ubah kode, lalu jalankan `docker compose up -d --build app`.

**Aplikasi lewat `go run`** (butuh Go 1.26 atau lebih baru). Cara ini lebih cepat untuk mencoba perubahan. Docker hanya menjalankan database dan penyimpanan berkas:

```bash
docker compose up -d postgres garage
cp .env.example .env              # PowerShell: Copy-Item .env.example .env
go run cmd/main.go --migrate:run --seed --run
```

Jangan menjalankan layanan `app` Docker bersamaan, karena sama-sama memakai port 8888.

Jalankan tes dengan `go test ./...`. File migrasi ada di `database/migrations/` dan dijalankan otomatis setiap aplikasi start. Setelah menambah atau mengubah endpoint, perbarui `docs/openapi.yaml` (file ini ikut dibangun ke dalam aplikasi, jadi build ulang agar `/docs` ikut berubah).

Struktur folder:

```
cmd/          titik masuk aplikasi
modules/      satu folder per fitur (routes, controller, service, repository, dto)
database/     entitas, migrasi, dan data awal
middlewares/  autentikasi, otorisasi, batas ukuran request
pkg/          kode bersama (aturan akses, penyimpanan berkas, helper)
docker/       Dockerfile, konfigurasi nginx dan Garage
docs/         spesifikasi OpenAPI dan ERD
```

## Kalau ada masalah

| Gejala | Penyebab dan solusi |
|---|---|
| `error during connect ... dockerDesktopLinuxEngine` | Docker belum berjalan. Buka Docker Desktop, tunggu siap, ulangi perintahnya |
| `port is already allocated` atau `address already in use` | Port dipakai program lain. Ubah `NGINX_PORT`, `GOLANG_PORT`, `DB_PORT`, atau `S3_HOST_PORT` di `.env` |
| Aplikasi terus restart | Lihat `docker compose logs app`. Penyebab umum: `JWT_SECRET` kosong atau kurang dari 16 karakter |
| Aplikasi tidak bisa terhubung ke database setelah mengganti `DB_PASS` | Postgres masih memakai password lama di volumenya. Jalankan `docker compose down -v`, lalu `docker compose up -d --build` |
| Ingin mulai dari nol | `docker compose down -v` lalu `docker compose up -d --build` |
| Unggahan berkas ditolak | Berkas harus PDF, JPG, atau PNG dengan ukuran maksimal 10 MB |

## Dokumentasi lain

- Dokumentasi API: <http://localhost:81/docs> saat aplikasi berjalan, sumbernya [docs/openapi.yaml](docs/openapi.yaml)
- Tabel database dan alur status tiap modul: [docs/ERD.md](docs/ERD.md)
- Asumsi dan pertanyaan terbuka per modul (aturan yang belum pasti dari desain): [docs/catatan/](docs/catatan/)

Proyek ini berawal dari [Go Gin Clean Starter](https://github.com/Caknoooo/go-gin-clean-starter) oleh Caknoooo (lisensi MIT, lihat [LICENSE](LICENSE)).
