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

## Evidence Sebelum Perbaikan

### 1. Missing recipient menghasilkan pesan generik

**Command:** `go test ./internal/handlers -v -count=1`

Pesan error yang dihasilkan masih generik (`required field is missing`), bukan pesan yang menyebut `recipient_name` secara spesifik.

```text
=== RUN   TestCreateDisbursementHTTP/missing_recipient_is_informative
    disbursement_integration_test.go:123:
                Error Trace:    D:/Naufal/Starter Project/internal/handlers/disbursement_integration_test.go:123
                Error:          "required field is missing" does not contain "recipient_name"
                Test:           TestCreateDisbursementHTTP/missing_recipient_is_informative
```

### 2. `limit=0` menyebabkan panic, `limit=-1` diterima tanpa validasi

**Command:** `go test ./internal/handlers -v -count=1`

`limit=0` menyebabkan `runtime error: integer divide by zero` pada perhitungan `total_pages` dan Gin mengembalikan HTTP 500. Sementara `limit=-1` diterima sebagai HTTP 200 dengan metadata yang tidak konsisten.

```text
=== RUN   TestListDisbursementsHTTP/zero_limit

2026/08/19 19:55:18 [Recovery] 2026/08/19 - 19:55:18 panic recovered:
runtime error: integer divide by zero
```

### 3. Concurrent approval tidak atomik (5x run gagal konsisten)

**Command:** `go test ./internal/services -run TestUpdateStatusConcurrentApprovalIsAtomic -v -count=5`

Kelima run gagal secara konsisten: seluruh 10 approval request sukses, padahal maksimum hanya satu yang boleh sukses.

```text
=== RUN   TestUpdateStatusConcurrentApprovalIsAtomic
    concurrency_test.go:83:
                Error Trace:    D:/Naufal/Starter Project/internal/services/concurrency_test.go:83
                Error:          "10" is not less than or equal to "1"
                Test:           TestUpdateStatusConcurrentApprovalIsAtomic
                Messages:       only one concurrent approval may succeed
--- FAIL: TestUpdateStatusConcurrentApprovalIsAtomic (0.00s)
FAIL
FAIL    example.com/disbursement/internal/services      0.801s
```

### 4. Race detector tidak menangkap masalah (logical race, bukan memory race)

**Command:** `go test -race ./internal/services -run TestUpdateStatusConcurrentApprovalIsAtomic -v -count=5`

Assertion tetap gagal dengan 10 sukses pada setiap run, tetapi tidak ada warning `DATA RACE`. Ini menunjukkan bahwa masalahnya adalah **logical check-then-act race**: operasi repository (`FindByID` lalu `Update`) masing-masing sudah memakai mutex, tetapi rangkaian keduanya tidak atomik.

```text
=== RUN   TestUpdateStatusConcurrentApprovalIsAtomic
    concurrency_test.go:83:
                Error Trace:    D:/Naufal/Starter Project/internal/services/concurrency_test.go:83
                Error:          "10" is not less than or equal to "1"
                Test:           TestUpdateStatusConcurrentApprovalIsAtomic
                Messages:       only one concurrent approval may succeed
--- FAIL: TestUpdateStatusConcurrentApprovalIsAtomic (0.00s)
FAIL
FAIL    example.com/disbursement/internal/services      1.002s
```

## Yang Tidak Perlu Diubah

Jangan ubah signature fungsi di `internal/services/disbursement.go` dan `internal/models/`. Test kamu harus test kode yang diberikan, bukan kode yang sudah dimodifikasi.

Kamu **boleh** menambahkan:
- File test (`*_test.go`) di direktori manapun
- Implementasi mock untuk repository interface
- File helper untuk test setup