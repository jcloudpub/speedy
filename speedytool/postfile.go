package speedytool

import (
	"os"
	"fmt"
	"sync"
	"time"
	"bytes"
	"strconv"
	"net/http"
	"github.com/jcloudpub/speedy/imageserver/util/log"
	"github.com/jcloudpub/speedy/imageserver/util"
)

const (
	PATH_PREFIX_DIR = "loadtest/abc/tag_"
	PATH_PREFIX     = "loadtest/abc/key_"
)

func PostFileTestSpeedyConcurrency(imageserverAddr string, fileName string, numGoroutine int, partSize int) {
	log.Infof("[PostFileTestSpeedyConcurrency] ==== begin")
	file, err := os.Open(fileName)
	if err != nil {
		log.Errorf("[PostFileTestSpeedyConcurrency] open file %s err: %v", fileName, err)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Errorf("[PostFileTestSpeedyConcurrency] get file %s stat err: %v", fileName, err)
		return
	}

	fileSize := info.Size()
	log.Infof("file %s size: %v", fileName, fileSize)
	fileBody := make([]byte, fileSize)
	nread, err := file.Read(fileBody)
	if err != nil || int64(nread) != fileSize {
		log.Errorf("[PostFileTestSpeedyConcurrency] read %s nread:%v, fileSize:%v, err: %v", fileName, nread, fileSize, err)
		return
	} 

	partCount := int(fileSize/int64(partSize))
	partial := int(fileSize%int64(partSize))
	result := make([]int, numGoroutine)
	var wg sync.WaitGroup
	wg.Add(numGoroutine)
	begin := time.Now()

	for i := 0; i < numGoroutine; i++ {
		go postFileTestSpeedy(imageserverAddr, i, fileBody, partCount, partial, partSize, result, &wg)
	}
	wg.Wait()
	end := time.Now()
	duration := end.Sub(begin)

	success := true 
	for i := 0; i < numGoroutine; i++ {
		if result[i] != 1 {
			log.Errorf("result[%d] error", i)
			success = false
		}
	}

	if !success {
		log.Infof("[PostFileTestSpeedyConcurrency] error")
		return
	}
	totalSize := fileSize * int64(numGoroutine)
	bandWidth := (float64(totalSize)/duration.Seconds())/(1024*1024) //B to MB
	log.Infof("[PostFileTestSpeedyConcurrency] upload bandWidth: %v MB/s", bandWidth)
}

func postFileTestSpeedy(imageserverAddr string, index int, fileBody []byte, partCount int, partial int, partSize int, result []int, wg *sync.WaitGroup) {
	defer wg.Done()
	if partCount == 0 && partial == 0 {
		return
	}

	path := PATH_PREFIX_DIR + strconv.Itoa(index)
	if partCount == 0 && partial != 0 {
		//upload
		err := postFile(imageserverAddr, path, fileBody[0:partial], 0, 0, int64(partial), true)
		if err != nil {
			log.Errorf("[postFileTestSpeedy] postFile error: %v", err)
			return
		}
	}

	begin := 0
	end := 0
	for k := 0; k < partCount-1; k++ {
		begin = k * partSize 
		end = (k+1) * partSize
		err := postFile(imageserverAddr, path, fileBody[begin:end], k, int64(begin), int64(end),  false)
		if err != nil {
			log.Errorf("[postFileTestSpeedy] postFile err: %v", err)
			return
		}
	}

	k := partCount - 1 
	begin = k * partSize
	if partial != 0 {
		end = begin + partial
	} else {
		end = (k+1)* partSize
	}
	err := postFile(imageserverAddr, path, fileBody[begin:end], k, int64(begin), int64(end),  true)
	if err != nil {
		log.Errorf("[postFileTestSpeedy] postFile err: %v", err)
		return
	}
	result[index] = 1
}

func postFile(imageserverAddr string, path string, data []byte, index int,  begin int64, end int64, isLast bool) error {
	header := make(map[string][]string)
	header["Path"] = []string{path}
	header["Fragment-Index"] = []string{fmt.Sprintf("%v", index)}
	header["Bytes-Range"] = []string{fmt.Sprintf("%v-%v", begin, end)}
	header["Is-Last"] = []string{fmt.Sprintf("%v", isLast)}

	_, statusCode, err := util.Call("POST", imageserverAddr, "/v1/file", bytes.NewBuffer(data), header )
	if err != nil || statusCode != http.StatusOK {
		return fmt.Errorf("[postFile] failed, path:%s, error:%v, statusCode:%v", path, err, statusCode)
	}
	log.Infof("[postFile] success, index:%v, path:%s", index, path)
	return nil
}

