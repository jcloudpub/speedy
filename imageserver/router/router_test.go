package router

import (
	"bytes"
	"github.com/jcloudpub/speedy/imageserver/util"
	"net/http"
	"sync"
	"testing"
)

var (
	path string = "repositories/username/ubuntu/tag_v2"
)

func TestRouterPostFile(t *testing.T) {
	s := "hello world content"

	header := make(map[string][]string, 0)
	header["Path"] = []string{path}
	header["Fragment-Index"] = []string{"0"}
	header["Bytes-Range"] = []string{"0-19"}
	header["Is-Last"] = []string{"true"}

	result, statusCode, err := util.Call("POST", "http://127.0.0.1:6788", "/v1/file", bytes.NewBuffer([]byte(s)), header)
	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
}

func postFile(t *testing.T, w *sync.WaitGroup) {
	defer w.Done()

	s := "hello world content"

	header := make(map[string][]string, 0)
	header["Path"] = []string{path}
	header["Fragment-Index"] = []string{"0"}
	header["Bytes-Range"] = []string{"0-19"} //length(s) == 19
	header["Is-Last"] = []string{"true"}

	result, statusCode, err := util.Call("POST", "http://127.0.0.1:6788", "/v1/file", bytes.NewBuffer([]byte(s)), header)
	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: %s", err)
	}

	if err != nil {
		t.Errorf("error: %v", err)
	}

	t.Log("result: %s\n", string(result))
}

func TestRouterPostFileConcurrent(t *testing.T) {
	t.Log("begin")
	var w sync.WaitGroup

	for i := 0; i < 20; i++ {
		w.Add(1)
		go postFile(t, &w)
	}
	w.Wait()
	t.Log("end")
}

func TestRouterGetFileInfo(t *testing.T) {
	header := make(map[string][]string, 0)
	header["Path"] = []string{path}

	result, statusCode, err := util.Call("GET", "http://127.0.0.1:6788", "/v1/fileinfo", nil, header)
	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
}

func TestRouterGetDirectoryInfo(t *testing.T) {
	header := make(map[string][]string, 0)
	header["Path"] = []string{"repositories/username/ubuntu"}

	result, statusCode, err := util.Call("GET", "http://127.0.0.1:6788", "/v1/list_directory", nil, header)

	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
}

func TestRouterDeleteMetaInfo(t *testing.T) {
	header := make(map[string][]string, 0)
	header["Path"] = []string{"testpath"}

	result, statusCode, err := util.Call("DELETE", "http://127.0.0.1:6788", "/v1/file", nil, header)

	if statusCode != http.StatusNoContent {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
}

func TestRouterGetFile(t *testing.T) {
	header := make(map[string][]string, 0)
	header["Path"] = []string{path}
	//header["Path"] = []string{"images/b70ad18cfc2aaad8c46a09cbe95c7e59938d27c2f06454a6bc052649e5442c24/_checksum"}
	header["Fragment-Index"] = []string{"0"}
	header["Bytes-Range"] = []string{"0-19"} //length("hello world")
	header["Is-Last"] = []string{"false"}

	result, statusCode, err := util.Call("GET", "http://127.0.0.1:6788", "/v1/file", nil, header)

	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
}

func TestRouterGetFileCurrent(t *testing.T) {
	t.Log("TestRouterGetFileCurrent begin")
	var w sync.WaitGroup

	for i := 0; i < 20; i++ {
		w.Add(1)
		go GetFileCurrent(t, &w)
	}
	w.Wait()
	t.Log("TestRouterGetFileCurrent end")
}

func GetFileCurrent(t *testing.T, w *sync.WaitGroup) {
	t.Log("begin == GetFileCurrent")

	header := make(map[string][]string, 0)
	header["Path"] = []string{path}
	header["Fragment-Index"] = []string{"0"}
	header["Bytes-Range"] = []string{"0-19"}
	header["Is-Last"] = []string{"true"}

	t.Log("GetFileCurrent === 2")

	result, statusCode, err := util.Call("GET", "http://127.0.0.1:6788", "/v1/file", nil, header)

	t.Log("GetFileCurrent === 3")

	if statusCode != http.StatusOK {
		t.Errorf("statusCode error: %d", statusCode, ", error: ", err)
	}

	if err != nil {
		t.Errorf("error: ", err)
	}

	t.Log("result: %s\n", string(result))
	w.Done()
}
