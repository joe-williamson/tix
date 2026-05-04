.PHONY: build install test fmt lint clean

BIN := tix
PKG := ./cmd/tix

build:
	go build -ldflags="-s -w" -o $(BIN) $(PKG)

install:
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
