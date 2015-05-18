all: build  

build: build-imageserver build-chunkmaster build-chunktool

build-imageserver:
	go build -o bin/imageserver ./cmd/imageserver

build-chunkmaster:
	go build -o bin/chunkmaster ./cmd/chunkmaster

build-chunktool:
	go build -o bin/chunktool ./cmd/chunktool

clean:
	@rm -rf bin

test:
	go test ./...
