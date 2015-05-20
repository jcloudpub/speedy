package main

import (
	"github.com/jcloudpub/speedy/imageserver/router"
	"flag"
	"strconv"
	"runtime"
	"os"
	"github.com/jcloudpub/speedy/imageserver/util/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var masterAddr = flag.String("mh", "127.0.0.1", "chunkmaster addr")
	var masterPort = flag.Int("mp", 8099, "chunkmaster port")
	var host = flag.String("h", "0.0.0.0", "imageserver listen ip")
	var port = flag.Int("p", 6788, "imageserver listen port")
	var limitNum = flag.Int("n", 2, "the limit num of available chunkserver each chunkserver group")
	var metaHost = flag.String("meta", "127.0.0.1;3306;root;;metadb", "meta server config eg. for mysql meta host;port;user;passwd;db")
	var debug = flag.Bool("D", false, "log debug level")

	flag.Parse()

	if *debug {
		os.Setenv("DEBUG", "DEBUG")
	}

	var masterUrl = "http://" + *masterAddr + ":" + strconv.Itoa(*masterPort)
	var imageServerAddr = *host + ":" + strconv.Itoa(*port)
	log.Infof("master URL: %s", masterUrl)
	log.Infof("listen: %s", imageServerAddr)

	log.Infof("the limit num of available chunkserver: %d", *limitNum)

	server := router.NewServer(masterUrl, *host, *port, *limitNum, *metaHost)
	log.Infof("start")

	err := server.Run()
	if err != nil {
		log.Errorf("start error: %v", err)
		return
	}
}
