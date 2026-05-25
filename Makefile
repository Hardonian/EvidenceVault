.PHONY: fmt vet test build smoke
fmt:
	gofmt -w ./cmd ./internal
vet:
	go vet ./...
test:
	go test ./...
build:
	go build ./cmd/server
smoke:
	bash scripts/smoke.sh
