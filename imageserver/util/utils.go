package util

import (
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
)

func Call(method, baseUrl, path string, body io.Reader, headers map[string][]string) ([]byte, int, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, baseUrl+path, body)
	if err != nil {
		return nil, 408, err
	}

	if headers != nil {
		for k, v := range headers {
			req.Header[k] = v
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return nil, resp.StatusCode, err
		}
		return nil, http.StatusNotFound, err
	}

	dataBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, resp.StatusCode, err
	}
	return dataBody, resp.StatusCode, nil
}


func GetRequestJsonParam(r *http.Request) (map[string]interface {}, error) {
	data, err := ioutil.ReadAll(r.Body)

	defer r.Body.Close()

	if err != nil {
		return nil, err
	}

	m, err := DecodeJson(data)

	if err != nil {
		return nil, err
	}

	return m, nil
}


func DecodeJson(data []byte) (map[string]interface {}, error) {
	var m map[string]interface {}

	err := json.Unmarshal(data, &m)

	if err != nil {
		return nil, err
	}

	return m, nil
}
