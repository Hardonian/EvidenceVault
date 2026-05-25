.PHONY: fmt vet test build smoke
fmt:
	@test -z "$(gofmt -l ./cmd ./internal)" || (echo 'gofmt required'; gofmt -l ./cmd ./internal; exit 1)
vet:
	go vet ./...
test:
	go test ./...
build:
	go build ./cmd/server
smoke:
	bash scripts/smoke.sh
