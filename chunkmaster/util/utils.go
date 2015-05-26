package util

import (
	"io"
	"fmt"
	"mime"
	"strconv"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"crypto/rand"
	"encoding/hex"

	"github.com/jcloudpub/speedy/chunkmaster/util/log"
)

func EncodeJson(data interface {}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func DecodeJson(data []byte) (map[string]interface {}, error) {
	var m map[string]interface {}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Call(method, baseUrl, path string, body io.Reader, headers map[string][]string)([]byte, int, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, baseUrl + path, body)
	if err != nil {
		return nil, 408, err
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers != nil {
		for k, v := range headers {
			req.Header[k] = v
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	dataBody, err:= ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return dataBody, resp.StatusCode, nil
}

func Response(data []byte, statusCode int, resp http.ResponseWriter) {
	resp.WriteHeader(statusCode)
	fmt.Fprintf(resp, string(data))
}

func NotFoundHandle(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(resp, "Not Found this page.")
}

func HandleError(resp http.ResponseWriter, respData string, err error, respCode int) {
	if err != nil {
		log.Errorf("resp err. %v", err)
	}

	log.Errorf("respCode %d.", respCode)
	resp.WriteHeader(respCode)

	log.Errorf("respData %s.", respData)
	fmt.Fprintf(resp, respData)
}

func ContentTypeCheck(r *http.Request) error {
	ct := r.Header.Get("Content-Type")

	// No Content-Type header is ok as long as there's no Body
	if ct == "" {
		if r.Body == nil || r.ContentLength == 0 {
			return nil
		}
	}

	// Otherwise it better be json
	if MatchsContentType(ct, "application/json") {
		return nil
	}

	return fmt.Errorf("Content-Type spcified (%s) must be 'application/json'", ct)
}

func MatchsContentType(contentType, expectedType string) bool {
	mimetype, _, err := mime.ParseMediaType(contentType)

	if err != nil {
		fmt.Errorf("Error parsing media type: %s error: %s", contentType, err.Error())
	}

	return err == nil && mimetype == expectedType
}

func truncateID(id string) string {
	shortLen := 12
	if len(id) < shortLen {
		shortLen = len(id)
	}
	return id[:shortLen]
}

// GenerateRandomID returns an unique id
func GenerateRandomID() string {
	for {
		id := make([]byte, 16)
		if _, err := io.ReadFull(rand.Reader, id); err != nil {
			panic(err) // This shouldn't happen
		}
		value := hex.EncodeToString(id)
		// if we try to parse the truncated for as an int and we don't have
		// an error then the value is all numberic and causes issues when
		// used as a hostname. ref #3869
		if _, err := strconv.ParseInt(truncateID(value), 10, 32); err == nil {
			continue
		}
		return value
	}
}

func CheckMapString(m map[string]interface{}, key string) (string, error) {
	if _, ok := m[key]; !ok {
		return "", fmt.Errorf(key + " is nil")
	}

	if v, ok := m[key].(string); ok {
		return v, nil
	}

	return "", fmt.Errorf(key + " is not string type")
}

func CheckMapInt64(m map[string]interface{}, key string) (int64, error) {
	if _, ok := m[key]; !ok {
		return 0, fmt.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return int64(v), nil
	}

	return 0, fmt.Errorf(key + " is not int64 type")
}

func CheckMapInt(m map[string]interface{}, key string) (int, error) {
	if _, ok := m[key]; !ok {
		return 0, fmt.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return int(v), nil
	}

	return 0, fmt.Errorf(key + " is not int type")
}

func CheckMapUInt16(m map[string]interface{}, key string) (uint16, error) {
	if _, ok := m[key]; !ok {
		return 0, fmt.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return uint16(v), nil
	}

	return 0, fmt.Errorf(key + " is not uint16 type")
}

func CheckMapUInt32(m map[string]interface{}, key string) (uint32, error) {
	if _, ok := m[key]; !ok {
		return 0, fmt.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return uint32(v), nil
	}

	return 0, fmt.Errorf(key + " is not uint32 type")
}
