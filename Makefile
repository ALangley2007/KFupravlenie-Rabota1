BINARY := bin/vshell

.PHONY: all build run test check demo clean

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

demo:
	sh scripts/test_options.sh
	sh scripts/test_startup.sh
	sh scripts/test_errors.sh

clean:
	rm -rf bin
