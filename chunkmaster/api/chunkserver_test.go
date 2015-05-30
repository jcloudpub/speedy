package api

import (
	"bytes"
	"github.com/jcloudpub/speedy/utils"
	"net/http"
	"testing"
)

const (
	HTTP_SERVER_HOST = "http://127.0.0.1:8099"
)

func TestChunkmasterRouteHandle(t *testing.T) {
	respData, respCode, err := util.Call("GET", HTTP_SERVER_HOST, "/v1/chunkmaster/route", bytes.NewBuffer([]byte(`""`)), nil)
	if err != nil || respCode != http.StatusOK {
		t.Log("TestChunkmasterRouteHandle Error")
		t.Fatal(err)
		return
	}
	t.Log("respData", string(respData))
}

func TestChunkmasterFidHandle(t *testing.T) {
	respData, respCode, err := util.Call("GET", HTTP_SERVER_HOST, "/v1/chunkmaster/fid", bytes.NewBuffer([]byte(`""`)), nil)
	if err != nil || respCode != http.StatusOK {
		t.Log("TestChunkmasterRouteHandle Error")
		t.Fatal(err)
		return
	}
	t.Log("respData: ", string(respData))
}

func TestChunkserverInitServerHandler(t *testing.T) {
	param := make(map[string]interface{})
	param["GroupId"] = 1
	param["Ip"] = "127.0.0.1"
	param["Port"] = 6666

	json, err := util.EncodeJson(param)
	if err != nil {
		t.Error(err)
	}

	respData, respCode, err := util.Call("POST", HTTP_SERVER_HOST, "/v1/chunkserver/initserver", bytes.NewBuffer(json), nil)
	if err != nil || respCode != http.StatusOK {
		t.Errorf("TestChunkserverInitServerHandler error: %v, respCode: %d", err, respCode)
	}

	t.Log("respData: %v", string(respData))
}

func TestReportChunkserverInfoHandler(t *testing.T) {
	param := make(map[string]interface{})
	param["GroupId"] = 1
	param["Ip"] = "127.0.0.1"
	param["Port"] = 6666
	param["TotalFreeSpace"] = 22234
	param["MaxFreeSpace"] = 23233
	param["PendingWrites"] = 10
	param["WritingCount"] = 12
	param["DataDir"] = "/export"
	param["ReadingCount"] = 1
	param["TotalChunks"] = 8
	param["ConnectionsCount"] = 4

	json, err := util.EncodeJson(param)
	if err != nil {
		t.Error(err)
	}

	respData, respCode, err := util.Call("POST", HTTP_SERVER_HOST, "/v1/chunkserver/reportinfo", bytes.NewBuffer(json), nil)
	if err != nil || respCode != http.StatusOK {
		t.Errorf("TestReportChunkserverInfoHandler error: %v, respCode: %d", err, respCode)
	}
	t.Log("reportChunkserverInfo success, respData: %v", respData)
}
