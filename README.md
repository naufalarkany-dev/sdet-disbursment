# Disbursement API - SDET Assignment

REST API sederhana untuk membuat, melihat, dan menyetujui disbursement. Repository ini berisi unit, HTTP integration, dan concurrency tests yang dirancang untuk membuktikan behavior penting serta menangkap defect pada starter implementation.

## Prerequisites

- Go 1.21 atau lebih baru
- GNU Make (opsional; semua target dapat dijalankan sebagai `go test` command)

Install atau rapikan dependency:

```sh
go mod tidy
```

## Menjalankan Aplikasi

```sh
make run
```

Server berjalan di `http://localhost:8080`. Secret JWT default adalah `test-secret-key` dan dapat diganti melalui `JWT_SECRET`.

Semua endpoint memakai `Authorization: Bearer <token>`. Token HS256 harus memiliki claim `sub` berupa username dan `role` berupa `admin` atau `operator`.

| Method | Path | Role |
|---|---|---|
| `POST` | `/disbursements` | Semua role |
| `PATCH` | `/disbursements/:id/status` | Admin |
| `GET` | `/disbursements` | Semua role |

## Menjalankan Test

```sh
make test              # seluruh test
make test-unit         # unit dan concurrency test pada service package
make test-integration  # HTTP integration test pada handler package
make test-race         # seluruh test dengan Go race detector
make coverage          # coverage.out dan ringkasan per function
go test ./internal/services -run '^$' -bench CalculateAdminFee -benchmem
```

Equivalent commands tanpa Make:

```sh
go test ./... -v -count=1
go test ./internal/services/... -v -count=1
go test ./internal/handlers/... -v -count=1
go test -race ./... -count=1
go test ./... -coverprofile=coverage.out -count=1
go tool cover -func=coverage.out
```

## Test Strategy

Unit tests memakai `testify/mock` untuk mengisolasi business logic dari persistence dan memverifikasi interaksi repository. HTTP integration tests memakai `net/http/httptest`, middleware JWT asli, serta repository memory baru pada setiap subtest agar data tidak bocor antar-case. Concurrency test memakai sepuluh goroutine, start gate, result channel, dan synchronization decorator di sekitar repository memory agar seluruh worker benar-benar membaca state `PENDING` sebelum mencoba update; test-side result collection tidak berbagi mutable state.

Boundary yang diuji mencakup fee threshold, minimum amount, final-state transitions, malformed JSON, authorization, filter, search, dan pagination invalid/extreme. Assertion memeriksa error identity atau message content, bukan hanya status code dan line coverage.

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

## Root Cause dan Business Risk

Starter service menjalankan `FindByID`, status validation, dan `Update` sebagai operasi terpisah, sehingga mutex pada masing-masing repository method tidak membuat keseluruhan check-then-act sequence atomik. Beberapa approver dapat membaca `PENDING` yang sama dan semuanya menganggap transisinya valid. Dalam sistem pembayaran, kondisi ini dapat menghasilkan multiple approval audit events, side effect ganda, atau disbursement yang diproses lebih dari sekali. Perbaikan menambahkan atomic compare-and-set capability pada repository memory: hanya update pertama yang status tersimpannya masih sesuai expected status dapat berhasil, sedangkan contender lain menerima `disbursement already in final state`.

## Evidence Setelah Perbaikan

### 1. Missing recipient menghasilkan pesan generik

Setelah perbaikan, pesan error sudah menyebut field yang bermasalah secara spesifik (`recipient_name`), bukan lagi pesan generik `required field is missing`.

**Command:**
```sh
go test ./internal/handlers -run "TestCreateDisbursementHTTP/missing_recipient_is_informative" -v -count=1
```

```text
=== RUN   TestCreateDisbursementHTTP
=== RUN   TestCreateDisbursementHTTP/missing_recipient_is_informative
--- PASS: TestCreateDisbursementHTTP (0.00s)
    --- PASS: TestCreateDisbursementHTTP/missing_recipient_is_informative (0.00s)
PASS
ok      example.com/disbursement/internal/handlers      1.073s
```

### 2. `limit=0` menyebabkan panic, `limit=-1` diterima tanpa validasi

- Default pagination mengembalikan `page=1` dan `limit=10`.
- `status=PENDING` dan case-insensitive `search=Budi` memfilter data dan total secara benar.
- `limit=0` dan `limit=-1` sekarang ditolak dengan HTTP 422 dan pesan `limit must be between 1 and 100` sebelum perhitungan `total_pages`.
- `page=999999` valid dan mengembalikan list kosong tanpa panic, dengan total keseluruhan tetap tersedia.
- `total_pages` sekarang dihitung dengan integer `int64` setelah limit tervalidasi, sehingga tidak ada division by zero atau metadata limit yang berbeda dari limit query.
- Case `total_pages_rounds_up` membuktikan total tiga item dengan `limit=2` menghasilkan `total_pages=2`.

**Command:**
```sh
go test ./internal/handlers -run TestListDisbursementsHTTP -v -count=1
```

Relevant output:
```text
=== RUN   TestListDisbursementsHTTP
=== RUN   TestListDisbursementsHTTP/default_pagination
=== RUN   TestListDisbursementsHTTP/pending_status_filter
=== RUN   TestListDisbursementsHTTP/recipient_search
=== RUN   TestListDisbursementsHTTP/zero_limit
=== RUN   TestListDisbursementsHTTP/negative_limit
=== RUN   TestListDisbursementsHTTP/page_far_beyond_results
--- PASS: TestListDisbursementsHTTP (0.00s)
    --- PASS: TestListDisbursementsHTTP/default_pagination (0.00s)
    --- PASS: TestListDisbursementsHTTP/pending_status_filter (0.00s)
    --- PASS: TestListDisbursementsHTTP/recipient_search (0.00s)
    --- PASS: TestListDisbursementsHTTP/zero_limit (0.00s)
    --- PASS: TestListDisbursementsHTTP/negative_limit (0.00s)
    --- PASS: TestListDisbursementsHTTP/page_far_beyond_results (0.00s)
PASS
ok      example.com/disbursement/internal/handlers      1.200s
```

### 3. Race condition pada concurrency approval

Focused verification concurrency dijalankan dengan race detector:

**Command:**
```sh
go test -race ./internal/services -run TestUpdateStatusConcurrentApprovalIsAtomic -v -count=5
```

Relevant output:

```text
=== RUN   TestUpdateStatusConcurrentApprovalIsAtomic
--- PASS: TestUpdateStatusConcurrentApprovalIsAtomic (0.00s)
PASS
ok      example.com/disbursement/internal/services      1.890s
```

Test lulus lima kali dan tidak ada warning dari race detector. HTTP integration tests juga lulus setelah missing field menyebut nama field dan invalid limit ditolak dengan HTTP 422.

## Most Valuable Tests

- `TestUpdateStatusConcurrentApprovalIsAtomic` menangkap regression atomicity yang tidak bisa ditemukan hanya dengan race detector atau sequential tests.
- `TestListDisbursementsHTTP` menangkap panic dan metadata pagination yang inkonsisten pada invalid boundary.
- `TestCreateDisbursementHTTP/missing_recipient_is_informative` memastikan client menerima error yang actionable, bukan sekadar HTTP 422.
- Repository failure unit tests memastikan error dipropagasikan dan caller tidak menerima object creation yang belum tersimpan.

## AI Assistance and Verification

AI digunakan untuk membantu reconnaissance, menyusun test cases, mengimplementasikan test harness, menganalisis output, dan mengusulkan minimal fixes. Setiap assertion diverifikasi terhadap business contract dan dijalankan langsung pada repository ini. Validitas test dibuktikan dengan red-before-fix dan green-after-fix, focused repeated concurrency runs, real JWT middleware, repository memory asli untuk integration tests, serta race detector; tidak ada output atau angka coverage yang dibuat-buat.

## Continuous Integration

`.github/workflows/test.yml` menjalankan `make test` dan `make test-race` pada setiap push dan pull request menggunakan versi Go dari `go.mod`. Workflow baru benar-benar berjalan setelah repository di-push ke GitHub; workflow tidak dieksekusi oleh perubahan lokal ini.

## Remaining Limitations

- Atomic compare-and-set saat ini diimplementasikan oleh repository memory melalui optional capability. Implementasi persistence baru harus menyediakan capability setara; fallback interface lama mempertahankan compatibility tetapi tidak menjamin atomicity lintas process.
- Integration tests tidak mencakup expired token, invalid signature, atau seluruh variasi malformed query karena berada di luar mandatory scope.
- Repository memory menggunakan map sehingga urutan hasil list tidak deterministik; tests sengaja memeriksa content/count contract, bukan urutan yang tidak dijanjikan API.
