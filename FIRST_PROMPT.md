# First Prompt for the OpenCode SDET Agent

Use the `sdet-assignment` agent and work inside the root of the provided `starter-project`.

Read the entire repository first, including `README.md`, `go.mod`, service, repository interface, in-memory repository, handlers, middleware, router/entry point, models, and Makefile. Pay special attention to `middleware/auth.go` to understand the exact mechanism used to determine admin vs non-admin role (JWT claim, header, context, etc.) before writing any request helper that needs to simulate different roles. Then implement the complete SDET coding assignment described below.

Your goal is a submission-quality test suite that proves meaningful behavior and exposes real defects—not merely high coverage.

Required scope:

1. Unit tests with `testify` and repository mocks for:
   - `CalculateAdminFee`: at least 0, 9,999, 4,999,999, 5,000,000, 10,000,000, and `math.MaxInt64`.
   - `ValidateStatusTransition`: pending→approved, pending→rejected, approved→approved, rejected→approved, pending→pending, and pending→empty status.
   - `DisbursementService.Create`: happy path and correct fee, missing recipient, missing account number, amount 9,999, negative amount with a specific non-misleading error, and repository-create failure with no partial state leaked to the caller.
   - `DisbursementService.UpdateStatus`: happy path, not found, already-final status, and repository-update failure.

2. HTTP integration tests with `net/http/httptest` and a fresh real in-memory repository per test for:
   - `POST /disbursements`: valid request=201, required-field error=422 with informative message, below-minimum amount=422, malformed JSON=400. For every error case, assert on the actual error message content, not just the status code.
   - `PATCH /disbursements/:id/status`: pending→approved=200, missing ID=404, repeated approved update=422 with `already in final state`, non-admin=403. Simulate roles using the real auth mechanism found during reconnaissance.
   - `GET /disbursements`: default pagination, `status=PENDING`, `search=Budi`, `limit=0`, `limit=-1`, and `page=999999`.

3. A concurrency test using at least ten goroutines that truly start together and approve the same pending disbursement. Collect results without introducing a test-side race. The required invariant is at most one success; all others must fail clearly. Run it with the race detector.

4. A root Makefile with working `test`, `test-unit`, `test-integration`, `test-race`, and `coverage` targets matching the assignment. Point `test-unit`/`test-integration` at the actual package paths used in this repository — do not copy example paths verbatim if the real structure differs.

5. A strong README containing run instructions, relevant `go test -race` output, whether the original concurrency assertion passed or failed, 3–5 sentence root-cause/business-risk analysis, pagination boundary findings, the most valuable tests, AI-use disclosure, verification method, and remaining limitations.

6. After all mandatory work: fix the concurrency defect without changing existing public repository-interface method signatures; prove red-before-fix and green-after-fix. Also investigate and test `total_pages`, add a `testing.B` benchmark for `CalculateAdminFee`, and add a GitHub Actions workflow (`.github/workflows/test.yml`) that runs `make test` and `make test-race` on push/PR. Note that you can create the workflow file and verify its YAML locally, but the workflow only actually executes once I push it to GitHub myself, since you never push.

Critical execution rules:

- Start with reconnaissance and baseline commands. Do not modify production code yet. Before doing anything else, check `git status` and `.gitignore`: if the repository has no baseline commit yet, or `.gitignore` is missing/incomplete (coverage.out, build binaries, etc. should be excluded), tell me immediately and wait — I will create the baseline commit myself, since you must never commit or push.
- Write the defect-revealing tests against the starter implementation first.
- Run focused tests and preserve truthful red evidence before applying any production fix.
- Run the concurrency test multiple times (e.g. `-count=5` or more) while iterating to confirm the race window triggers consistently, not just once, before treating the result as final evidence.
- Notify me at each natural commit checkpoint so I can commit manually (you must never run `git commit` or `git push` yourself): (a) once required tests are written and red evidence against the unmodified starter code is captured and documented, tell me this is a good point to commit red evidence; (b) once minimal fixes are applied and the suite is green with README updated, tell me this is a good point to commit the fix. Don't just silently proceed past these points.
- Once all mandatory items (1–5) are done and verified green, stop and report status to me before starting any bonus item.
- Assertions must express the correct contract; never weaken them to accept broken behavior.
- Differentiate a logical check-then-act race from a Go memory data race. Report race-detector output exactly as observed.
- Keep public signatures intact and avoid unrelated refactors.
- Do not commit or push.
- Do not fabricate test output or coverage.
- At every major phase, briefly tell me what you found and what command actually ran.
- At the end, run `go vet ./...`, `make test`, `make test-race`, `make coverage`, `git diff --check`, and `git status --short`, then summarize results and teach me the key points I need to explain during the interview.

Begin now with repository reconnaissance and baseline verification. If the current directory is not the starter-project root, locate it first. Only stop to ask me if the starter project is genuinely missing or the code cannot be executed after reasonable diagnosis.
