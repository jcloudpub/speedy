package main

import (
	"fmt"
	"flag"
	"runtime"
	"github.com/jcloudpub/speedy/speedytool"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var imageserverIp = flag.String("h", "127.0.0.1", "imageserver ip")
	var imageserverPort = flag.Int("p", 6788, "imageserver port")
	var fileName = flag.String("f", "test", "file used to upload, e.g. /tmp/test")
	var numGoroutine = flag.Int("n", 1, "num of goroutine")
	var partSizeMB = flag.Int("s", 4, "the part size(MB)")

	flag.Parse()
	imageserverAddr := fmt.Sprintf("http://%s:%d", *imageserverIp, *imageserverPort)
	partSize := *partSizeMB * 1024 *1024
	speedytool.TestSpeedyConcurrency(imageserverAddr, *fileName, *numGoroutine, partSize)
}


