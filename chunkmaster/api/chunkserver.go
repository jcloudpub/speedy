package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jcloudpub/speedy/chunkmaster/metadata"
	"github.com/jcloudpub/speedy/logs"
	"github.com/jcloudpub/speedy/utils"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	MAX_RANGE      = 10000
	ALLOCATE_RANGE = MAX_RANGE / 100

	INIT_STATUS = 0
	RW_STATUS   = 1
	RO_STATUS   = 2
	ERR_STATUS  = 3

	GLOBAL_NORMAL_STATUS  = 0
	GLOBAL_TRANSFER_STAUS = 8
)

func MonitorTicker(intervalSecond int, timeoutSecond int) {
	timer := time.NewTicker(time.Duration(intervalSecond) * time.Second)
	for {
		select {
		case <-timer.C:
			chunkserverMonitor(timeoutSecond)
		}
	}
}

func chunkserverMonitor(timeoutSecond int) {
	serverInfoAbnormal := make(map[string]*metadata.Chunkserver)
	now := time.Now()
	timeout := time.Duration(timeoutSecond) * time.Second

	lock.RLock()

	for key, chunkserver := range serverInfo {
		if RW_STATUS == chunkserver.Status {
			if now.Sub(chunkserver.UpdateTime) > timeout {
				serverInfoAbnormal[key] = chunkserver
			}
		}
	}
	lock.RUnlock()

	if len(serverInfoAbnormal) == 0 {
		return
	}

	for key, chunkserver := range serverInfoAbnormal {
		err := mdDriver.UpdateChunkserverStatus(chunkserver, RW_STATUS, RO_STATUS)
		if err != nil {
			log.Errorf("[ChunkServerMonitor] update chunkserver failed: %v, %v", chunkserver, err)
		} else {
			updateChunkserverInfo(key, RO_STATUS)
		}
	}
}

func updateChunkserverInfo(key string, status int) {
	lock.Lock()
	chunkserver, ok := serverInfo[key]
	if !ok {
		lock.Unlock()
		log.Errorf("[updateServerInfo] chunkserver: %v do not exist", key)
		return
	}

	chunkserver.Status = status
	lock.Unlock()
	log.Errorf("[updateServerInfo] update RW_STATUS to RO_STATUS chunkserver: %v", key)
}

func reportChunkserverInfoHandler(resp http.ResponseWriter, req *http.Request) {
	reqData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	log.Debugf("[reportChunkserverInfoHandler] reqData: %v", string(reqData))
	var jsonMap map[string]interface{}
	err = json.Unmarshal(reqData, &jsonMap)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}

	chunkserver, err := metadata.GenChunkserver(jsonMap)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("%d:%s:%d", chunkserver.GroupId, chunkserver.Ip, chunkserver.Port)
	log.Debugf("key: %v", key)
	log.Debugf("serverInfo: %v", serverInfo)

	lock.RLock()
	oldChunkserver, ok := serverInfo[key]
	if !ok {
		lock.RUnlock()
		util.HandleError(resp, "", fmt.Errorf("not exist: %v ", chunkserver), http.StatusBadRequest)
		return
	}
	lock.RUnlock()

	err = reportChunkserverInfo(key, chunkserver, oldChunkserver)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
	}

	log.Debugf("[reportInfoHandler] update chunkserver success: %v", chunkserver)
	util.Response(nil, http.StatusOK, resp)
}

func reportChunkserverInfo(key string, chunkserver *metadata.Chunkserver, oldChunkserver *metadata.Chunkserver) error {
	if oldChunkserver.Status == INIT_STATUS {
		err := mdDriver.UpdateChunkserverInfo(chunkserver, INIT_STATUS, RW_STATUS)
		if err != nil {
			return err
		}
		chunkserver.Status = RW_STATUS
	}

	if RW_STATUS == oldChunkserver.Status {
		err := mdDriver.UpdateChunkserverInfo(chunkserver, RW_STATUS, RW_STATUS)
		if err != nil {
			return err
		}
		chunkserver.Status = RW_STATUS
	}

	if RO_STATUS == oldChunkserver.Status {
		err := mdDriver.UpdateChunkserverInfo(chunkserver, RO_STATUS, RW_STATUS)
		if err != nil {
			return err
		}
		log.Infof("[reportChunkserverInfo] update RO_STATUS to RW_STATUS chunkserver: %v", key)
		chunkserver.Status = RW_STATUS
	}

	chunkserver.UpdateTime = time.Now()

	lock.Lock()
	_, ok := serverInfo[key]
	if !ok {
		lock.Unlock()
		return fmt.Errorf("do not exist: %v", chunkserver)
	}
	serverInfo[key] = chunkserver
	lock.Unlock()
	return nil
}

func initChunkserverHandler(resp http.ResponseWriter, req *http.Request) {
	reqData, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	log.Infof("[initserverHandler] read reqData %v", string(reqData))

	var jsonMap map[string]interface{}
	err = json.Unmarshal(reqData, &jsonMap)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	log.Infof("[initserverHandler] change json to map %v", jsonMap)

	groupId := uint16((jsonMap["GroupId"]).(float64))
	ip := jsonMap["Ip"].(string)
	port := int((jsonMap["Port"]).(float64))
	chunkserver := new(metadata.Chunkserver)
	chunkserver.GroupId = groupId
	chunkserver.Ip = ip
	chunkserver.Port = port
	chunkserver.Status = INIT_STATUS
	chunkserver.TotalFreeSpace = 0
	chunkserver.MaxFreeSpace = 0
	chunkserver.PendingWrites = 0
	chunkserver.WritingCount = 0
	chunkserver.DataDir = ""
	chunkserver.ReadingCount = 0
	chunkserver.TotalChunks = 0
	chunkserver.ConnectionsCount = 0

	err = addChunkserver(chunkserver)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}

	util.Response(nil, http.StatusOK, resp)
}

func batchInitChunkserverHandler(resp http.ResponseWriter, req *http.Request) {
	reqData, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	log.Infof("[batchInitserverHandler] read reqData %v", string(reqData))

	var chunkserverList []metadata.Chunkserver
	err = json.Unmarshal(reqData, &chunkserverList)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	log.Infof("[batchInitserverHandler] change json to arr %v", chunkserverList)

	err = batchAddChunkserver(&chunkserverList)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}

	util.Response(nil, http.StatusOK, resp)
}

func loadChunkserverInfoHandler(resp http.ResponseWriter, req *http.Request) {
	err := LoadChunkserverInfo()
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}
	log.Infof("[loadChunkserverInfoHandler] load chunkserver info success")
	util.Response(nil, http.StatusOK, resp)
}

func chunkserverGroupInfoHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	groupIdStr, ok := vars["groupId"]
	if !ok {
		util.HandleError(resp, "", fmt.Errorf("groupId is empty"), http.StatusBadRequest)
		return
	}

	groupId, err := strconv.ParseInt(groupIdStr, 10, 0)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}

	result, err := chunkserverGroupInfo(int(groupId))
	if err != nil {
		util.HandleError(resp, "", err, http.StatusBadRequest)
		return
	}
	util.Response(result, http.StatusOK, resp)
}

func chunkserverGroupInfo(groupId int) ([]byte, error) {
	chunkserverGroup, err := mdDriver.ListChunkserverGroup(groupId)
	if err != nil {
		return nil, err
	}

	chunkserverGroupJsonByte, err := json.Marshal(chunkserverGroup)
	if err != nil {
		return nil, err
	}

	return chunkserverGroupJsonByte, nil
}

func LoadChunkserverInfo() error {
	chunkserverList, err := mdDriver.ListChunkserver()
	if err != nil {
		return err
	}
	now := time.Now()
	serverInfoTemp := make(map[string]*metadata.Chunkserver)
	for _, server := range chunkserverList {
		if server.Status == ERR_STATUS {
			continue
		}

		key := fmt.Sprintf("%d:%s:%d", server.GroupId, server.Ip, server.Port)
		server.UpdateTime = now
		serverInfoTemp[key] = server
		log.Infof("[loadChunkserverInfo] chunkserver: %v", server)
	}

	lock.Lock()
	serverInfo = serverInfoTemp
	lock.Unlock()
	return nil
}

func addChunkserver(chunkserver *metadata.Chunkserver) error {
	chunkserver.Status = INIT_STATUS
	chunkserver.TotalFreeSpace = 0
	chunkserver.MaxFreeSpace = 0
	chunkserver.PendingWrites = 0
	chunkserver.WritingCount = 0
	chunkserver.DataDir = ""
	chunkserver.ReadingCount = 0
	chunkserver.TotalChunks = 0
	chunkserver.ConnectionsCount = 0

	err := mdDriver.AddChunkserver(chunkserver)
	if err != nil {
		return err
	}
	lock.Lock()
	defer lock.Unlock()

	key := fmt.Sprintf("%d:%s:%d", chunkserver.GroupId, chunkserver.Ip, chunkserver.Port)
	chunkserver.UpdateTime = time.Now()
	serverInfo[key] = chunkserver
	return nil
}

func batchAddChunkserver(chunkserverList *[]metadata.Chunkserver) error {
	for _, chunkserver := range *chunkserverList {
		err := addChunkserver(&chunkserver)
		if err != nil {
			return err
		}
	}
	return nil
}

func chunkserverCheckError(resp http.ResponseWriter, req *http.Request) {
	chunkservers, err := mdDriver.ListChunkserver()
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}

	existErrChunkserver := false
	for _, chunkserver := range chunkservers {
		if chunkserver.Status == ERR_STATUS {
			existErrChunkserver = true
			break
		}
	}

	respData := []byte("0")
	if existErrChunkserver {
		respData = []byte("1")
	}

	util.Response(respData, http.StatusOK, resp)
}

func chunkmasterRouteHandler(resp http.ResponseWriter, req *http.Request) {
	lock.RLock()

	chunkserverGroup := make(map[string]metadata.Chunkservers)
	for _, chunkserver := range serverInfo {
		groupId := fmt.Sprintf("%v", chunkserver.GroupId)

		list, ok := chunkserverGroup[groupId]
		if !ok {
			list = make(metadata.Chunkservers, 0, 3)
		}

		chunkserverGroup[groupId] = append(list, chunkserver)
	}

	lock.RUnlock()

	respData, err := json.Marshal(chunkserverGroup)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	util.Response(respData, http.StatusOK, resp)
}

func chunkmasterFidHandler(resp http.ResponseWriter, req *http.Request) {
	fidBegin, fidEnd, err := allocFid()
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}
	log.Debugf("[chunkmasterFidHandle] allocate Fid fidBegin %v, fidEnd %v", fidBegin, fidEnd)

	jsonMap := make(map[string]interface{})
	jsonMap["FidBegin"] = fidBegin
	jsonMap["FidEnd"] = fidEnd
	respData, err := json.Marshal(jsonMap)
	if err != nil {
		util.HandleError(resp, "", err, http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	util.Response(respData, http.StatusOK, resp)
}

func allocFid() (uint64, uint64, error) {
	fid.Lock()
	defer fid.Unlock()

	var (
		fidBegin uint64
		fidEnd   uint64
		err      error
	)

	if fid.Begin == fid.End {
		fid.Begin, err = mdDriver.GetFid()

		if err != nil {
			return 0, 0, err
		}

		fid.End = fid.Begin + MAX_RANGE
		err = mdDriver.UpdateFid(fid.End)
		if err != nil {
			fid.End = 0
			fid.Begin = 0
			return 0, 0, err
		}
	}

	fidBegin = fid.Begin
	fidEnd = fid.Begin + ALLOCATE_RANGE
	fid.Begin = fidEnd

	return fidBegin, fidEnd, nil
}
