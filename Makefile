all: build  

build: build-imageserver build-chunkmaster build-chunktool build-chunkserver build-speedytool

build-imageserver:
	go build -o bin/imageserver ./cmd/imageserver

build-chunkmaster:
	go build -o bin/chunkmaster ./cmd/chunkmaster

build-chunkserver:
	make -C chunkserver
	@cp -f chunkserver/spy_server ./bin/spy_server

build-chunktool:
	go build -o bin/chunktool ./cmd/chunktool

build-speedytool:
	go build -o bin/speedytool ./cmd/speedytool

clean:
	@rm -rf bin
	@rm -rf chunkserver/*.o
	@rm -rf chunkserver/spy_server

test:
	go test ./...
