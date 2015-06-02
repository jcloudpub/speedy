package api

import (
	"net/http"
)

var RouteMap map[string]map[string]http.HandlerFunc

func init() {
	RouteMap = map[string]map[string]http.HandlerFunc{
		"POST": {
			"/v1/chunkserver/batchinitserver": batchInitChunkserverHandler,
			"/v1/chunkserver/initserver":      initChunkserverHandler,
			"/v1/chunkserver/reloadinfo":      loadChunkserverInfoHandler,
			"/v1/chunkserver/reportinfo":      reportChunkserverInfoHandler,
		},
		"GET": {
			"/v1/chunkmaster/route": chunkmasterRouteHandler,
			"/v1/chunkmaster/fid":   chunkmasterFidHandler,

			"/v1/chunkserver/{groupId}/groupinfo": chunkserverGroupInfoHandler,
			"/v1/chunkserver/checkerror":          chunkserverCheckError,
		},
	}
}
