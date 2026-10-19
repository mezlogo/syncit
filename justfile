all: format clean build

clean:
	rm -rf bin/

test:
	go test ./...

format:
	gofmt -w .

build:
	go build -o bin/ ./...

release:
	go build -ldflags="-s -w" -trimpath -o bin/ ./...

compress:
	./upx --best bin/syncit
