package main

import (
	"flag"
	"github.com/jcloudpub/speedy/imageserver/router"
	"github.com/jcloudpub/speedy/logs"
	"os"
	"runtime"
	"strconv"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var masterIp = flag.String("mh", "127.0.0.1", "chunkmaster ip")
	var masterPort = flag.Int("mp", 8099, "chunkmaster port")
	var host = flag.String("h", "0.0.0.0", "imageserver listen ip")
	var port = flag.Int("p", 6788, "imageserver listen port")
	var limitNum = flag.Int("n", 2, "the limit num of available chunkserver each chunkserver group")
	var metaIp = flag.String("dh", "127.0.0.1", "metadb ip")
	var metaPort = flag.Int("dp", 3306, "metadb port")
	var userName = flag.String("u", "root", "metadb user")
	var password = flag.String("pw", "", "metadb password")
	var metadb = flag.String("db", "metadb", "meta database")
	var debug = flag.Bool("D", false, "log debug level")
	var connPoolCapacity = flag.Int("c", 200, "the capacity of every chunkserver's connection pool")

	flag.Parse()

	if *debug {
		os.Setenv("DEBUG", "DEBUG")
	}

	var masterUrl = "http://" + *masterIp + ":" + strconv.Itoa(*masterPort)
	log.Infof("master URL: %s", masterUrl)
	log.Infof("the limit num of available chunkserver: %d", *limitNum)

	server := router.NewServer(masterUrl, *host, *port, *limitNum, *metaIp, *metaPort, *userName, *password, *metadb, *connPoolCapacity)
	log.Infof("imageserver start...")
	err := server.Run()
	if err != nil {
		log.Errorf("start error: %v", err)
		return
	}
}
