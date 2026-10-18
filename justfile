all: format clean build

clean:
	rm -rf bin/

test:
	go test ./...

format:
	gofmt -w .

build:
	go build -o bin/ ./...
