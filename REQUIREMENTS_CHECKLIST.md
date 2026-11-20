# REQUIREMENTS CHECKLIST — PABWE 2026 Praktikum 9

Legenda: ✅ = sudah dijalankan & lulus di sandbox pembuatan (hasil command nyata) · 🟡 = diimplementasi, **belum bisa diverifikasi** di sandbox (butuh PostgreSQL/Jenkins/OpenAI/browser) — jalankan sendiri · ⚠️ = asumsi/catatan.

Isi ZIP: 4 repo — `*-latihan-p9-be-golang-gin` + `*-latihan-p9-fe-reactjs` (**Todolist Pro**, port 8080/5173) dan `*-sk-p9-be-golang-gin` + `*-sk-p9-fe-reactjs` (**Delcom Blog**, port 8081/5174). Dua aplikasi dipisah, tidak dicampur.

## Hasil verifikasi aktual
| Item | Todolist Pro | Delcom Blog |
|---|---|---|
| `go vet ./...`, `gofmt` | ✅ | ✅ |
| `go test ./...` (termasuk `cmd/server`) | ✅ lulus | ✅ lulus |
| Go statement coverage (`go tool cover -func`, total) | ✅ **100.0%** | ✅ **100.0%** |
| `npm run lint` | ✅ | ✅ |
| Vitest | ✅ 102 tes | ✅ 93 tes |
| FE coverage statements/branches/functions/lines | ✅ **100/100/100/100** | ✅ **100/100/100/100** |
| `npm run build` (Vite) | ✅ | ✅ |
| Go embed + SPA fallback dengan `dist` asli (uji `httptest`) | ✅ (rute `/author`, `/api/*` benar) | ✅ |

⚠️ Backend diuji dengan SQLite in-memory (sesuai modul) — di sandbox memakai driver `mattn/go-sqlite3` karena `modernc.org` (dependensi `glebarez/sqlite`) tidak dapat diunduh; kode yang dikirim memakai `github.com/glebarez/sqlite` sesuai modul (perbedaannya hanya 1 import). Jalankan `go mod tidy` pada mesin Anda (go.sum tidak disertakan). Query admin/pencarian memakai SQL yang kompatibel PostgreSQL, tetapi **belum dijalankan terhadap PostgreSQL 17 sungguhan** 🟡.

## A. Umum (kedua aplikasi)
- ✅ Prefix `/api`; format `{status,message,data}` (`success` 2xx, `fail` 4xx, `error` 5xx); handler `NoRoute` JSON 404; recovery panic terpusat.
- ✅ Clean Architecture + Feature-First: `internal/features/<fitur>/{dto,repository,service,handler}`, DI manual via konstruktor, middleware terpisah, config/database/router/server terpisah.
- ✅ JWT HS256 (expiry wajib, metode dibatasi), bcrypt, role dari token bertanda tangan (bukan dari klien), CORS (`CORS_ALLOWED_ORIGINS`), header keamanan, rate limit (login/register 20/mnt, AI 10/mnt), request timeout.
- ✅ IDOR: semua query data milik pengguna memakai `user_id` (diuji 404 untuk pengguna lain).
- ✅ Swagger UI `/swagger/index.html`, spesifikasi `/api.json` (host/skema dinamis), `docs/api.json|swagger.json|swagger.yaml` (24 / 20 endpoint). ⚠️ Dihasilkan skrip, bukan `swag init`.
- ✅ Frontend: React Router, protected route, ID login `login-username|login-password|login-submit`, modal **full-screen** (header `bg-*-700`, footer gelap, Batal & Simpan di kanan), kebab menu ⋮ (Detail/Edit/Hapus + konfirmasi hapus), loading/error/empty/success toast, statistik dari API, daftar berbentuk tabel.
- ✅ Embed: `internal/web` (`//go:embed all:dist`), `scripts/build.sh` (npm build → salin ke `internal/web/dist` → `go build`). `/api`, `/swagger`, `/api.json` tidak tertangkap SPA; aset hilang → 404; refresh `/author`, `/admin`, `/posts/:slug` → `index.html`.
- 🟡 Binary smoke test (`scripts/smoke.sh`) butuh PostgreSQL — belum dijalankan.
- 🟡 Jenkinsfile (checkout, setup, PG17 via docker, gofmt/vet, test + cek 100%, FE lint/test/coverage, build, embed, binary, smoke, arsip). Gagal otomatis bila gate tidak terpenuhi. **Belum dijalankan di Jenkins**; sesuaikan `FE_REPO_URL` dan credential `openai-key`.
- 🟡 Tampilan visual desktop/mobile belum dilihat di browser sungguhan (hanya diuji via jsdom + class Tailwind). Video demo: belum.

## B. Todolist Pro (latihan)
| Requirement | Status |
|---|---|
| Register/Login/Logout/Profile/Update Profile | ✅ |
| CRUD todo, toggle status (PATCH), detail, filter `todo_group_id`/`status`/`q` | ✅ |
| TodoGroup CRUD + `is_public` | ✅ |
| Template publik (guest & member lihat), **Gunakan Template** = kloning kelompok + todo jadi data baru milik pengguna dalam satu transaksi (status direset, data asli tidak berubah) | ✅ |
| AI Task Planner (OpenAI chat completions via backend; validasi JSON; timeout→504, upstream→502, belum dikonfigurasi→503; preview → terapkan ke kelompok) — diuji dengan mock HTTP | ✅ · 🟡 tidak dicoba ke OpenAI asli |
| Admin: daftar pengguna + statistik todo, todo per pengguna, hanya role admin | ✅ |
| Statistik `total`, `total_selesai`, `belum_selesai` dari DB | ✅ |
| Seeder: `admin/admin`, `member/member` | ✅ |
| Rute FE: `/` (template publik), `/templates/:id`, `/login`, `/register`, `/dashboard`, `/todos/:id`, `/admin` | ✅ |
⚠️ Modul menyebut `search.go` pada struktur latihan; fitur latihan tidak memerlukannya, jadi tidak dibuat.

## C. Delcom Blog (studi kasus)
| Requirement | Status |
|---|---|
| Endpoint `/api/public/{categories,posts,posts/:slug}`, `/api/auth/*`, `/api/posts` (+`/stats`, `/:id`, `PATCH /:id/toggle-publish`, `/ai-generate`), `/api/admin/{users,posts,users/:id/posts}` | ✅ |
| Model User/Post: slug unik, kategori (Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum), is_published, published_at, FK cascade | ✅ |
| Draf tidak terbaca guest (404 di API publik) | ✅ |
| Estimasi waktu baca, info penulis (tanpa email) | ✅ |
| Author hanya mengelola artikel sendiri; admin hanya monitoring (tidak bisa CRUD artikel) | ✅ |
| **AI Blog Generator**: topik → **Search Engine Service** → sumber → OpenAI → draf `{title,category,excerpt,content,sources,searched}`; sumber = hasil search nyata (URL http(s) tervalidasi), sumber karangan AI diabaikan; jika search gagal → error 502 (tidak pura-pura); `include_web_search=false` → `searched:false` tanpa sumber | ✅ (mock HTTP) |
| ⚠️ Format respons Search Engine tidak ditentukan modul: klien menerima array, `{"results":[…]}` atau `{"data":[…]}` dengan `title`, `url`/`link`, `snippet`/`content`/`description`; `GET {SEARCH_ENGINE_ENDPOINT}?q=` dengan `Authorization: Bearer $OPENAI_KEY`. **Cocokkan dengan endpoint Delcom asli.** | 🟡 |
| TOAST UI Editor (markdown, preview, mode gelap/terang, sinkron state, `destroy()` saat unmount) & Viewer (sanitasi bawaan) | ✅ (dengan mock pada tes) · 🟡 render asli di browser |
| Publik: navbar sticky, tanpa sidebar, pencarian real-time, filter kategori, `/posts/:slug`; `/author`; `/admin` | ✅ |
| Seeder: `admin/admin`, `author/author`, `member/member` (role author) ⚠️ modul menulis role "member" pada satu tempat; akun `member` diberi role `author` agar dapat menulis | ✅ |

## D. Keamanan
✅ bcrypt · ✅ JWT kedaluwarsa/signature/alg-none ditolak · ✅ IDOR · ✅ LIKE di-escape (tanpa SQL injection; semua query berparameter) · ✅ pesan error 5xx generik · ✅ API key hanya di backend · ✅ cover image hanya `http(s)://` atau path `/…` · ✅ rate limit · ⚠️ rate limiter in-memory per proses (tanpa pembersihan bucket lama) · ⚠️ `npm audit` melaporkan kerentanan pada dependensi `@toast-ui/editor` (SK) — periksa sebelum produksi.
