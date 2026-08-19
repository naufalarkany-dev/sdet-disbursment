# Disbursement API — SDET Starter Project

Ini adalah starter project untuk **Coding Test SDET**. Kode di sini sudah berjalan — tugasmu adalah menulis test-nya.

## Struktur Project

```
├── cmd/main.go                          # Entry point aplikasi
├── internal/
│   ├── models/disbursement.go           # Data model & request/response types
│   ├── repository/disbursement.go       # Interface repository (gunakan ini untuk mock di unit test)
│   ├── repository/memory/               # Implementasi in-memory (gunakan untuk integration test)
│   ├── services/disbursement.go         # Business logic — INI yang perlu kamu test
│   ├── handlers/disbursement.go         # HTTP handler — untuk integration test
│   └── middleware/auth.go               # JWT middleware
├── Makefile
└── go.mod
```

## Prerequisites

- Go 1.21+
- `make` (opsional, bisa run command secara manual)

## Setup

```bash
go mod tidy
```

## Menjalankan Aplikasi

```bash
make run
# atau: go run cmd/main.go
```

Server berjalan di `http://localhost:8080`. JWT secret default: `test-secret-key`.

## Endpoints

| Method | Path | Deskripsi | Auth |
|---|---|---|---|
| `POST` | `/disbursements` | Buat disbursement baru | JWT (semua role) |
| `PATCH` | `/disbursements/:id/status` | Update status (APPROVED/REJECTED) | JWT (role: admin) |
| `GET` | `/disbursements` | List disbursement dengan filter & pagination | JWT (semua role) |

## Autentikasi

Semua endpoint memerlukan JWT di header `Authorization: Bearer <token>`.

JWT harus berisi claim:
- `sub`: username (string)
- `role`: `"admin"` atau `"operator"`

Contoh generate token untuk testing (Go):

```go
import "github.com/golang-jwt/jwt/v4"

token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "sub":  "admin",
    "role": "admin",
    "exp":  time.Now().Add(24 * time.Hour).Unix(),
})
tokenString, _ := token.SignedString([]byte("test-secret-key"))
```

## Menjalankan Test

```bash
make test              # semua test
make test-unit         # unit test service layer saja
make test-integration  # integration test handler saja
make coverage          # semua test + coverage report
```

> **Catatan:** Saat kamu submit, direktori ini harus sudah berisi file test yang kamu tulis di `internal/services/` dan `internal/handlers/`.

## Yang Tidak Perlu Diubah

Jangan ubah signature fungsi di `internal/services/disbursement.go` dan `internal/models/`. Test kamu harus test kode yang diberikan, bukan kode yang sudah dimodifikasi.

Kamu **boleh** menambahkan:
- File test (`*_test.go`) di direktori manapun
- Implementasi mock untuk repository interface
- File helper untuk test setup
