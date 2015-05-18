package main

import (
	"fmt"
	"os"
	"flag"
	"runtime"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/jcloudpub/speedy/chunkmaster/api"
	"github.com/jcloudpub/speedy/chunkmaster/util"
	"github.com/jcloudpub/speedy/chunkmaster/util/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var serverPort = flag.Uint("p", 8099, "chunkmaster server port")
	var metaHost = flag.String("h", "127.0.0.1", "db ip")
	var metaPort = flag.String("m", "3306", "db port")
	var user = flag.String("u", "root", "db user")
	var passwd = flag.String("w", "", "db passwd")
	var db = flag.String("d", "speedy", "db name")
	var debug = flag.Bool("D", false, "log debug level")

	flag.Parse()

	api.InitAll(*metaHost, *metaPort, *user, *passwd, *db)

	//set log debug level
	if *debug {
		setLogDebugLevel()
	}

	err := api.LoadChunkserverInfo()
	if err != nil {
		log.Errorf("loadChunkserverInfo error: %v", err)
		os.Exit(-1)
	}

	go api.MonitorTicker(5, 30)

	router := initRouter()
	http.Handle("/", router)
	log.Infof("listening in port %d", *serverPort)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), nil);err != nil {
		log.Errorf("listening err %v", err)
		os.Exit(-1)
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

func setLogDebugLevel() {
	os.Setenv("DEBUG", "DEBUG")
}
