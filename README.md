# Delcom Blog — Backend (Golang Gin)

Go 1.27 · Gin · GORM · PostgreSQL 17 · JWT · bcrypt · Swagger · OpenAI · Search Engine

## Menjalankan
```bash
cp .env.example .env        # isi OPENAI_KEY (dan OPENAI_MODEL)
docker compose up -d        # PostgreSQL 17 (atau pakai PostgreSQL lokal)
go mod tidy                 # go.sum belum disertakan
go run ./cmd/server         # http://localhost:8081
```
Swagger UI: http://localhost:8081/swagger/index.html · Spesifikasi: `/api.json`

## Test & coverage (target 100%)
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -1     # total: 100.0%
```
Unit test memakai SQLite in-memory (`glebarez/sqlite`, sesuai modul) dan mock HTTP untuk OpenAI dan Search Engine.

## Build produksi (frontend di-embed ke binary)
```bash
FE_DIR=../ifs24056-pabwe2026-sk-p9-fe-reactjs ./scripts/build.sh   # npm build -> internal/web/dist -> go build bin/server
./bin/server                         # satu binary: API + frontend (SPA fallback)
./scripts/smoke.sh                   # smoke test (butuh PostgreSQL aktif)
```
Akun seeder: `admin/admin`, `author/author`, `member/member` — ganti sebelum produksi. Lihat `REQUIREMENTS_CHECKLIST.md`.
