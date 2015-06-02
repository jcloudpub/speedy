package main

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/jcloudpub/speedy/chunkmaster/api"
	"github.com/jcloudpub/speedy/logs"
	"github.com/jcloudpub/speedy/utils"
	"net/http"
	"os"
	"runtime"
	"strconv"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var serverHost = flag.String("h", "0.0.0.0", "chunkmaster listen ip")
	var serverPort = flag.Int("p", 8099, "chunkmaster listen port")
	var metaHost = flag.String("dh", "127.0.0.1", "database ip")
	var metaPort = flag.String("dp", "3306", "database port")
	var user = flag.String("u", "root", "database user")
	var passwd = flag.String("pw", "", "database passwd")
	var db = flag.String("d", "speedy", "database name")
	var debug = flag.Bool("D", false, "log debug level")

	flag.Parse()

	api.InitAll(*metaHost, *metaPort, *user, *passwd, *db)

	//set log debug level
	if *debug {
		os.Setenv("DEBUG", "DEBUG")
	}

	err := api.LoadChunkserverInfo()
	if err != nil {
		log.Fatalf("loadChunkserverInfo error: %v", err)
	}

	go api.MonitorTicker(5, 30)

	router := initRouter()
	http.Handle("/", router)
	log.Infof("listen %s:%d", *serverHost, *serverPort)

	if err := http.ListenAndServe(*serverHost+":"+strconv.Itoa(*serverPort), nil); err != nil {
		log.Fatalf("listen error: %v", err)
	}
}

func initRouter() *mux.Router {
	router := mux.NewRouter()

	log.Debugf("initRouter")

	for method, routes := range api.RouteMap {
		for route, fct := range routes {
			localRoute := route
			localMethod := method
			log.Debugf("route: %s, method: %v", route, method)
			router.Path(localRoute).Methods(localMethod).HandlerFunc(fct)
		}
	}

	router.NotFoundHandler = http.HandlerFunc(util.NotFoundHandle)
	return router
}
