.PHONY: build run test lint clean

BINARY := bin/multica-mcp

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf bin/
