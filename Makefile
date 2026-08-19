.PHONY: test test-unit test-integration coverage run tidy

run:
	go run cmd/main.go

tidy:
	go mod tidy

test:
	go test ./... -v -count=1

test-unit:
	go test ./internal/services/... -v -count=1

test-integration:
	go test ./internal/handlers/... -v -count=1

coverage:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

lint:
	go vet ./...
