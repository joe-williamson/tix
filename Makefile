.PHONY: build install test fmt lint clean

BIN := tix
PKG := ./cmd/tix

build:
	go mod tidy
	go build -ldflags="-s -w" -o $(BIN) $(PKG)

install:
	go mod tidy
	go install $(PKG)

test:
	go test ./...

fmt:
	gofmt -w .
	goimports -w .

lint:
	golangci-lint run

clean:
	rm -f $(BIN)
