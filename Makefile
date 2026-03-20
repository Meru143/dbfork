.PHONY: build install test test-integration lint release-dry

build:
	go build -o bin/dbfork ./cmd/dbfork/

install:
	go install ./cmd/dbfork/

test:
	go test ./...

test-integration:
	go test ./test/integration/...

lint:
	golangci-lint run ./...

release-dry:
	goreleaser release --snapshot --clean
