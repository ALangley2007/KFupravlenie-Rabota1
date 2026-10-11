BINARY := bin/vshell

.PHONY: all build run test check clean

all: check test build

build:
	go build -o $(BINARY) ./src

run:
	go run ./src

test:
	go test ./...

check:
	gofmt -l src tests
	go vet ./...

clean:
	rm -rf bin
