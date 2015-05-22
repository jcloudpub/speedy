package router

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jcloudpub/speedy/imageserver/chunkserver"
	"github.com/jcloudpub/speedy/imageserver/meta"
	"github.com/jcloudpub/speedy/imageserver/util"
	_"github.com/jcloudpub/speedy/imageserver/meta/redisdriver"
	"github.com/jcloudpub/speedy/imageserver/meta/mysqldriver"
	"time"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"math/rand"
	"sync/atomic"
	"math"
	"github.com/jcloudpub/speedy/imageserver/util/log"
)

const (
	headerPath        = "Path"
	headerIndex       = "Fragment-Index"
	headerRange       = "Bytes-Range"
	headerIsLast      = "Is-Last"
	LimitCSNormalSize = 2
	SUCCESS           = ""
)

type Server struct {
	MasterUrl		  string
	Ip                string
	Port              int
	router            *mux.Router
	running           bool
	mu                sync.Mutex
	fids              *chunkserver.Fids                      //ChunkServerGoups
	chunkServerGroups *chunkserver.ChunkServerGroups         //groupId <> []ChunkServer
	connectionPools   *chunkserver.ChunkServerConnectionPool //{"host:port":connectionPool}
	metaDriver		  meta.MetaDriver
	limitNum		  int
	getFidRetryCount  int32
	metadbIp          string
	metadbPort		  int
	metadbUser		  string
	metadbPassword	  string
	metaDatabase	  string
}

func NewServer(masterUrl, ip string, port int, num int, metadbIp string, metadbPort int, metadbUser, metadbPassword, metaDatabase string) *Server {
	return &Server{
		MasterUrl:		   masterUrl,
		Ip:                ip,
		Port:              port,
		fids:              chunkserver.NewFids(),
		chunkServerGroups: nil,
		connectionPools:   nil,
		limitNum:		   num,
		getFidRetryCount:  0,
		metadbIp:          metadbIp,
		metadbPort:		   metadbPort,
		metadbUser:		   metadbUser,
		metadbPassword:	   metadbPassword,
		metaDatabase:	   metaDatabase,
	}
}

func (s *Server) initApi() {
	m := map[string]map[string]http.HandlerFunc{
		"GET": {
			"/v1/fileinfo": s.getFileInfo,
			"/v1/file":     s.getFile,
			"/v1/list_directory": s.getDirectoryInfo,
		},
		"POST": {
			"/v1/file":  s.uploadFile,
			"/v1/_ping": s.ping,
		},
		"DELETE": {
			"/v1/file": s.deleteFile,
		},
	}

	s.router = mux.NewRouter()
	for method, routes := range m {
		for route, fct := range routes {
			s.router.Path(route).Methods(method).HandlerFunc(fct)
		}
	}
	s.router.NotFoundHandler = http.NotFoundHandler()
}

func (s *Server) responseResult(data []byte, statusCode int, err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	log.Debugf("responseResult len: %d", len(data))
	w.WriteHeader(statusCode)
	log.Debugf("responseResult len: %d", len(string(data)))
	w.Write(data)
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	header := r.Header
	path := header.Get(headerPath)
	fragmentIndex := header.Get(headerIndex)
	bytesRange := header.Get(headerRange)
	isLast := header.Get(headerIsLast)

	start, end, err := s.splitRange(bytesRange)
	if err != nil {
		log.Errorf("splitRange error: %s", err)
		s.responseResult(nil, http.StatusBadRequest, err, w)
		return
	}

	last := false
	if isLast == "true" || isLast == "TRUE" {
		last = true
	}

	index, err := strconv.ParseUint(fragmentIndex, 10, 64)
	if err != nil {
		log.Errorf("parse fragmentIndex error: %s", err)
		s.responseResult(nil, http.StatusBadRequest, err, w)
		return
	}

	log.Infof("[postFile] path: %s, fragmentIndex: %d, bytesRange: %d-%d, isLast: %v", path, index, start, end, last)

	metaInfoValue := &meta.MetaInfoValue{
		Index:  index,
		Start:  start,
		End:    end,
		IsLast: last,
	}
	metaInfo := &meta.MetaInfo{Path: path, Value: metaInfoValue}

	//upload concurrently, select a group
	//chunkServers, err := s.getChunkServerGroup(int64(end-start), metaInfoValue)
	chunkServers, err := s.selectChunkServerGroupComplex(int64(end-start), metaInfoValue)
	if err != nil {
		log.Errorf("select ChunkServerGroup error: %s", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	if len(chunkServers) == 0 {//TODO need check, num of chunkserver >= 2
		log.Errorf("select ChunkServerGroup error, len(chunkServers) == 0")
		s.responseResult(nil, http.StatusInternalServerError, fmt.Errorf("select ChunkServerGroup error"), w)
		return
	}

	log.Debugf("ChunkServerGroup: %s", chunkServers)

	fileId, err := s.getFid()
	if err != nil {
		log.Errorf("get fileId error: %s", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("read request body error: %s", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	var rangeSize uint64
	rangeSize = end - start
	if len(data) != int(rangeSize) {
		log.Errorf("the data length is: %d, rangeSize is: %d", len(data), rangeSize)
		s.responseResult(nil, http.StatusBadRequest, fmt.Errorf("length of data and range are different"), w)
		return
	}

	log.Debugf("begin to post concurrently")

	var normal int = 0
	for i := 0; i < len(chunkServers); i++ {
		if chunkServers[i].Status == chunkserver.RW_STATUS {
			normal++
		}
	}

	//ch := make(chan string, len(chunkServers))
	ch := make(chan string, normal)
	for i := 0; i < len(chunkServers); i++ {
		if chunkServers[i].Status == chunkserver.RW_STATUS {
			go s.postFileConcurrence(&chunkServers[i], data, ch, fileId)
		}
	}

	log.Debugf("handle post result, num: %d", normal)
	err = s.handlePostResult(ch, normal)
	if err != nil {
		log.Errorf("post error: %s", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Debugf("post end")
	log.Debugf("begin to store metaInfo: %s", metaInfo)

	metaInfo.Value.FileId = fileId
	metaInfo.Value.GroupId = uint16(chunkServers[0].GroupId)

	//err = meta.StoreMetaInfo(metaInfo)
	err = s.metaDriver.StoreMetaInfo(metaInfo)
	if err != nil {
		log.Errorf("store metaInfo error: %s", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Infof("[postFile] success. path: %s, fragmentIndex: %d, bytesRange: %d-%d, isLast: %v", path, index, start, end, last)

	s.responseResult(nil, http.StatusOK, nil, w)
}

func (s *Server) getFileInfo(w http.ResponseWriter, r *http.Request) {
	path := r.Header.Get(headerPath)

	log.Infof("[getFileInfo] Path: %s", path)

	result, err := s.metaDriver.GetFileMetaInfo(path, false)
	if err != nil {
		log.Errorf("getFileInfo get metainfo error, key: %s, error: %s", path, err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	if len(result) == 0 {
		log.Infof("getFileInfo metainfo not exists, key: %s", path)
		s.responseResult(nil, http.StatusNotFound, err, w)
		return
	}

	resultMap := make(map[string]interface {})
	resultMap["fragment-info"] = result
	jsonResult, err := json.Marshal(resultMap)
	if err != nil {
		log.Errorf("json.Marshal error, key: %s, error: %s", path, err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Infof("[getFileInfo] success, path: %s, result: %s", path, string(jsonResult))

	s.responseResult(jsonResult, http.StatusOK, nil, w)
}

func (s *Server) getDirectoryInfo(w http.ResponseWriter, r *http.Request) {
	path := r.Header.Get(headerPath)

	log.Infof("[getDirectoryInfo] directory: %s", path)

	result, err := s.metaDriver.GetDirectoryInfo(path)
	if err != nil {
		log.Errorf("getDirectoryInfo get directory info error, directory: %s, error: %s", path, err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	if len(result) == 0 {
		log.Infof("directory is empty, path: %s", path)
		s.responseResult(nil, http.StatusNotFound, err, w)
		return
	}

	resultMap := make(map[string]interface {})
	resultMap["file-list"] = result
	jsonResult, err := json.Marshal(resultMap)
	if err != nil {
		log.Errorf("json.Marshal error, result: %s", jsonResult)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Infof("[getDirectoryInfo] success, directory: %s, result: %s", path, string(jsonResult))
	s.responseResult(jsonResult, http.StatusOK , nil, w)
}

func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	header := r.Header
	path := header.Get(headerPath)
	fragmentIndex := header.Get(headerIndex)
	bytesRange := header.Get(headerRange)
	start, end, err := s.splitRange(bytesRange)
	if err != nil {
		log.Errorf("splitRange error, bytesRange: %s, error: %s", bytesRange, err)
		s.responseResult(nil, http.StatusBadRequest, err, w)
		return
	}

	index, err := strconv.ParseUint(fragmentIndex, 10, 64)
	if err != nil {
		log.Errorf("parser fragmentIndex: %s, error: %s", fragmentIndex, err)
		s.responseResult(nil, http.StatusBadRequest, err, w)
		return
	}

	log.Infof("[getFile] path: %s, fragmentIndex: %d, bytesRange: %d-%d", path, index, start, end)

	metaInfoValue := &meta.MetaInfoValue{
		Index: index,
		Start: start,
		End:   end,
	}
	metaInfo := &meta.MetaInfo{Path: path, Value: metaInfoValue}
	log.Debugf("metaInfo: %s", metaInfo)

	chunkServer, err := s.getOneNormalChunkServer(metaInfo)
	if err != nil {
		if err.Error() == "fragment metainfo not found" {
			s.responseResult(nil, http.StatusNotFound, err, w)
		} else {
			s.responseResult(nil, http.StatusInternalServerError, err, w)
		}
		return
	}

	connPools := s.GetConnectionPools()
	conn, err := connPools.GetConn(chunkServer)
	log.Debugf("getFile getconnection success")
	if err != nil {
		log.Errorf("getFile getconnection error: %v", err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	data, err := chunkServer.GetData(metaInfo.Value, conn.(*chunkserver.PooledConn))
	if err != nil {
		conn.Close()
		connPools.ReleaseConn(conn)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Infof("[getFile] success. path: %s, fragmentIndex: %d, bytesRange: %d-%d", path, index, start, end)

	connPools.ReleaseConn(conn)

	w.Header().Set("Content-Type", "octet-stream")
	s.responseResult(data, http.StatusOK, nil, w)
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	s.responseResult([]byte("{OK}"), http.StatusOK, nil, w)
}

func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.Header.Get(headerPath)

	log.Infof("[deleteFile] path: %s", path)

	err := s.metaDriver.DeleteFileMetaInfo(path)
	if err != nil {
		log.Errorf("delete metainfo error for path: %s, error: %s", path, err)
		s.responseResult(nil, http.StatusInternalServerError, err, w)
		return
	}

	log.Infof("[deleteFile] success. path: %s", path)
	s.responseResult(nil, http.StatusNoContent, nil, w)
}

func (s *Server) splitRange(bytesRange string) (uint64, uint64, error) {
	var start, end uint64

	fmt.Sscanf(bytesRange, "%d-%d", &start, &end)
	if start >= end {
		return 0, 0, fmt.Errorf("bytesRange error!")
	}

	return start, end, nil
}

func (s *Server) handlePostResult(ch chan string, size int) error {
	var result, tempResult string
	var failed = false

	if ch == nil {
		log.Errorf("ch is nil")
		return fmt.Errorf("handlePostResult ch is nil")
	}

	log.Debugf("len(ch): %d, size: %d", len(ch), size)
	for k := 0; k < size; k++ {
		tempResult = <-ch
		if len(tempResult) != 0 {
			result = tempResult
			failed = true
		}
	}

	if failed {
		log.Errorf("handlePostResult failed: %s", result)
		return fmt.Errorf(result)
	}

	return nil
}

func (s *Server) getFid() (uint64, error){
	fileId, err := s.fids.GetFid()

	if err != nil {

		var count int32 = 1
		var init int32 = 0
		swapped := atomic.CompareAndSwapInt32(&s.getFidRetryCount, init, count)
		if !swapped {
			log.Infof("another goroutine is trying to get fid, waiting")
			filedId, err := s.fids.GetFidWait()
			if err != nil {
				return 0, err
			}
			return filedId, nil
		}

		log.Infof("=== try to get fid range === begin ===")
		defer atomic.CompareAndSwapInt32(&s.getFidRetryCount, count, init)

		err1 := s.GetFidRange(false)
		log.Infof("=== try to get fid range === end ===")

		if err1 != nil {
			log.Errorf("getFid try to get fid failed: %v", err1)
			return 0, err1
		}

		fileId, err1 = s.fids.GetFid()
		if err1 != nil {
			log.Errorf("getFid error: %v", err1)
			return 0, err1
		}
	}

	return fileId, nil
}

func (s *Server) postFileConcurrence(chunkServer *chunkserver.ChunkServer, data []byte, c chan string, fileId uint64) {
	log.Debugf("postFileConcurrence === begin to get connection")
	log.Debugf("chunkServer: %v", chunkServer)

	connPools := s.GetConnectionPools()
	if connPools == nil {
		log.Errorf("connectionPools is nil")
		c <- "connectionPools is nil"
		return
	}

	log.Debugf("fid is: %d", fileId)
	log.Debugf("connPools: %v, %s", connPools, connPools)

	conn, err := connPools.GetConn(chunkServer)
	log.Debugf("connection is: %v", conn)

	if err != nil {
		log.Errorf("can not get connection: %s", err.Error())
		c <- err.Error()
		return
	}

	log.Debugf("begin to upload data")
	err = chunkServer.PutData(data, conn.(*chunkserver.PooledConn), fileId)
	if err != nil {
		log.Errorf("upload data failed: %s", err)
		conn.Close()
		s.connectionPools.ReleaseConn(conn)
		c <- err.Error()
		return
	}

	log.Debugf("upload data success")
	c <- SUCCESS
	log.Debugf("set SUCCESS to chan")

	connPools.ReleaseConn(conn)
	log.Debugf("release connection success")
}

func (s *Server) getOneNormalChunkServer(mi *meta.MetaInfo) (*chunkserver.ChunkServer, error) {
	log.Debugf("getOneNormalChunkServer === begin")
	log.Debugf("metainfo: %s", mi)

	metaInfoValue, err := s.metaDriver.GetFragmentMetaInfo(mi.Path, mi.Value.Index, mi.Value.Start, mi.Value.End)
	if err != nil {
		log.Errorf("GetFragmentMetaInfo: %s, error: %s", mi, err)
		return nil, err
	}

	if metaInfoValue == nil {
		log.Errorf("fragment metainfo not found, path: %s, index: %d, start: %d, end: %d",
			mi.Path, mi.Value.Index, mi.Value.Start, mi.Value.End)
		return nil, fmt.Errorf("fragment metainfo not found")
	}

	mi.Value = metaInfoValue
	log.Debugf("getOneNormalChunkServer, metaInfo: %s", mi)
	log.Debugf("groupId :%d", mi.Value.GroupId)

	groupId := strconv.Itoa(int(mi.Value.GroupId))
	groups := s.GetChunkServerGroups()
	servers, ok := groups.GroupMap[groupId]
	if !ok {
		log.Errorf("getOneNormalChunkServer do not exist group: %s", groupId)
		return nil, fmt.Errorf("do not exist group: %s", groupId)
	}

	index := rand.Int() % len(servers)
	server := servers[index]
	if server.Status == chunkserver.RW_STATUS {
		log.Debugf("get an chunkserver: %s", server)
		return &server, nil
	}

	for i := 0; i < len(servers); i++ {
		server = servers[i]
		if server.Status == chunkserver.RW_STATUS {
			log.Debugf("get an chunkserver: %s", server)
			return &server, nil
		}
	}

	log.Errorf("can not find an available chunkserver, metainfo: %s", mi)
	return nil, fmt.Errorf("can not find an available chunkserver")
}


func (s *Server) selectChunkServerGroupSimple(size int64, meta *meta.MetaInfoValue) ([]chunkserver.ChunkServer, error) {
	//TODO get a normal group, the MaxFreeSpace should > size, and the health num >= LimitCSNormalSize
	//store processId and fileId to meta
	groups := s.GetChunkServerGroups()
	var resultGroupId string = ""

	for groupId, servers := range groups.GroupMap {
		var finded bool = true

		for index := 0; index < len(servers); index++ {
			server := servers[index]
			if server.MaxFreeSpace < size {
				finded = false
				break
			}
		}

		if finded {
			resultGroupId = groupId
		}
	}

	if resultGroupId != "" {
		return groups.GroupMap[resultGroupId], nil
	}

	return nil, fmt.Errorf("can not find an available chunkserver")
}

func (s *Server) selectChunkServerGroupComplex(size int64, meta *meta.MetaInfoValue) ([]chunkserver.ChunkServer, error) {
	if size <= 0 {
		log.Errorf("data size: %d <= 0")
		return nil, fmt.Errorf("data size: %d <= 0", size)
	}

	groups := s.GetChunkServerGroups()
	var totalNum int = len(groups.GroupMap)
	var selectNum int = totalNum/10 + 3
	minHeap := chunkserver.NewMinHeap(selectNum)

	for groupId, servers := range groups.GroupMap {
		var minMaxFreeSpace int64 = math.MaxInt64
		var normalNum int = 0
		var avilable = true
		var pendingWrites = 0
		var writingCount = 0

		length := len(servers)

		// skip empty group and transfering... group
		if length == 0 || servers[0].GlobalStatus != chunkserver.GLOBAL_NORMAL_STATUS {
			continue
		}

		for index := 0; index < length; index++ {
			server := servers[index]

			if server.Status != chunkserver.ERR_STATUS && server.Status != chunkserver.RW_STATUS {
				avilable = false
				break;
			}

			if server.Status == chunkserver.ERR_STATUS {
				continue
			}

			if server.Status == chunkserver.RW_STATUS {
				normalNum += 1
			}

			if server.MaxFreeSpace < minMaxFreeSpace {
				minMaxFreeSpace = server.MaxFreeSpace
			}

			if server.PendingWrites > pendingWrites {
				pendingWrites = server.PendingWrites
			}

			if server.WritingCount > writingCount {
				writingCount = server.WritingCount
			}
		}

		if avilable && minMaxFreeSpace > size && normalNum >= s.limitNum {
			minHeap.AddElement(groupId, minMaxFreeSpace, pendingWrites, writingCount)
		}
	}

	if minHeap.GetSize() < selectNum {
		selectNum = minHeap.GetSize()
	}

	if selectNum == 0 {
		log.Errorf("selectNum == 0, there's not an avaiable chunkserver")
		return nil, fmt.Errorf("there's not an avaiable chunkserver")
	}

	minHeap.BuildMinHeapSecondary()

	log.Debugf("minHeap: %s", minHeap)

	index := rand.Int()%selectNum
	log.Debugf("index: %d", index)
	resultGroupId, err := minHeap.GetElementGroupId(index)

	if err != nil {
		log.Errorf("can not find an available chunkserver: %s", err)
		return nil, fmt.Errorf("can not find an available chunkserver")
	}

	log.Debugf("resultGroupId: %s, chunkServers: %v", resultGroupId, groups.GroupMap[resultGroupId])
	return groups.GroupMap[resultGroupId], nil
}

func (s *Server) GetChunkServerInfo() error {
	byteData, statusCode, err := util.Call("GET", s.MasterUrl, "/v1/chunkmaster/route", nil, nil)
	if err != nil {
		log.Errorf("GetChunkServerInfo response code: %d", statusCode)
		log.Errorf("GetChunkServerInfo error: %s", err)
		return err
	}

	if statusCode != http.StatusOK {
		log.Errorf("response code: %d", statusCode)
		return fmt.Errorf("statusCode error: %d", statusCode)
	}

	infos := make(map[string][]chunkserver.ChunkServer)
	err = json.Unmarshal(byteData, &infos)
	if err != nil {
		log.Errorf("json.Unmarshal response data error: %s", err)
		return err
	}

	s.handleChunkServerInfo(infos)
	return nil
}

func (s *Server) GetFidRange(mergeWait bool) error {
	if !s.fids.IsShortage() {
		return nil
	}

	byteData, statusCode, err := util.Call("GET", s.MasterUrl, "/v1/chunkmaster/fid", nil, nil)
	if err != nil {
		log.Errorf("GetChunkServerInfo response code: %d, err: %s", statusCode, err)
		return err
	}

	if statusCode != http.StatusOK {
		log.Errorf("response code: %d", statusCode)
		return fmt.Errorf("statusCode error: %d", statusCode)
	}

	log.Infof("GetFidRange data: %s", string(byteData))

	newFids := chunkserver.NewFids()
	err = json.Unmarshal(byteData, &newFids)
	if err != nil {
		log.Errorf("GetFidRange json.Unmarshal response data error: %s", err)
		return err
	}

	log.Infof("get new fid: %s", newFids)
	s.fids.Merge(newFids.Start, newFids.End, mergeWait)
	return nil
}

func (s *Server) handleChunkServerInfo(infos map[string][]chunkserver.ChunkServer) {
	var (
		delServers []*chunkserver.ChunkServer
		addServers []*chunkserver.ChunkServer
	)

	newChunkServerGroups := &chunkserver.ChunkServerGroups{
		GroupMap: infos,
	}
	oldChunkServerGroups := s.GetChunkServerGroups()

	if oldChunkServerGroups == nil {
		delServers, addServers = serverInfoDiff(infos, nil)
	} else {
		delServers, addServers = serverInfoDiff(infos, oldChunkServerGroups.GroupMap)
	}

	if len(delServers) == 0 && len(addServers) == 0 {
		s.ReplaceChunkServerGroups(newChunkServerGroups)
		return
	}

	log.Infof("len(delServers): %d, delServers: %v", len(delServers), delServers)
	log.Infof("len(addServers): %d, addServers: %v", len(addServers), addServers)

	oldConnectionPool := s.GetConnectionPools()
	newConnectionPool := chunkserver.NewChunkServerConnectionPool()

	if oldConnectionPool != nil {
		log.Infof("oldConnectionPool: %v", oldConnectionPool)
		for key, connectionPool := range oldConnectionPool.Pools {
			newConnectionPool.AddExistPool(key, connectionPool)
		}
	}

	if len(delServers) != 0 {
		log.Infof("handleChunkServerInfo deleteServers: %s", delServers)
		for index := 0; index < len(delServers); index++ {
			newConnectionPool.RemovePool(delServers[index])
		}
	}

	if len(addServers) != 0 {
		log.Infof("handleChunkServerInfo addServes: %s", addServers)
		for index := 0; index < len(addServers); index++ {
			newConnectionPool.AddPool(addServers[index])
		}
	}

	log.Infof("newConnectionPool: %v", newConnectionPool)
	log.Infof("newChunkServerGroups: %v", newChunkServerGroups)

	s.ReplaceConnPoolsAndChunkServerGroups(newConnectionPool, newChunkServerGroups)

	if len(delServers) != 0 && oldConnectionPool != nil {
		log.Infof("handleChunkServerInfo deleteServers: %s", delServers)
		for index := 0; index < len(delServers); index++ {
			oldConnectionPool.RemoveAndClosePool(delServers[index])
		}
	}
}

func (s *Server) GetChunkServerGroups() *chunkserver.ChunkServerGroups {
	s.mu.Lock()
	groups := s.chunkServerGroups
	s.mu.Unlock()
	return groups
}

func (s *Server) GetConnectionPools() *chunkserver.ChunkServerConnectionPool {
	s.mu.Lock()
	connectionPool := s.connectionPools
	s.mu.Unlock()
	return connectionPool
}

func (s *Server) ReplaceChunkServerGroups(newGroups *chunkserver.ChunkServerGroups) {
	s.mu.Lock()
	s.chunkServerGroups = newGroups
	s.mu.Unlock()
}

func (s *Server) ReplaceConnPoolsAndChunkServerGroups(newConnectionPool *chunkserver.ChunkServerConnectionPool, newGroups *chunkserver.ChunkServerGroups) {
	s.mu.Lock()
	s.connectionPools = newConnectionPool
	s.chunkServerGroups = newGroups
	s.mu.Unlock()
}

func serverInfoDiff(newInfo, oldInfo map[string][]chunkserver.ChunkServer) (delServers, addServers []*chunkserver.ChunkServer) {
	addServers = infoDiff(newInfo, oldInfo)
	delServers = infoDiff(oldInfo, newInfo)

	log.Debugf("addServers: %v, deleteServers: %v", addServers, delServers)

	return delServers, addServers
}

//diff = info1 - (the intersection info1 and info2  )
func infoDiff(info1, info2 map[string][]chunkserver.ChunkServer) ([]*chunkserver.ChunkServer) {
	diffServers := make([]*chunkserver.ChunkServer, 0)

	for groupId, servers1 := range info1 {
		servers2, ok := info2[groupId]

		if !ok {
			for index := 0; index < len(servers1); index++ {
				diffServers = append(diffServers, &servers1[index])
			}

			continue;
		}

		for index1 := 0; index1 < len(servers1); index1++ {
			server1 := servers1[index1]
			found := false

			for index2 := 0; index2 < len(servers2); index2++ {
				server2 := servers2[index2]

				if server1.HostInfoEqual(&server2) {
					found = true
					break
				}
			}

			if !found {
				diffServers = append(diffServers, &server1)
			}
		}
	}

	return diffServers
}

func (s *Server) GetFidRangeTicker() {
	timer := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-timer.C:
			err := s.GetFidRange(true)
			if err != nil {
				log.Errorf("GetFidRange error: %v", err)
			}
		}
	}
}

func (s *Server) GetChunkServerInfoTicker() {
	timer := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-timer.C:
			err := s.GetChunkServerInfo()
			if err != nil {
				log.Errorf("GetChunkServerInfoTicker error: %s", err)
			}
		}
	}
}

func (s *Server) Run() error {
	log.Infof("begin run")

	s.initApi()
	err := s.GetChunkServerInfo()
	if err != nil {
		log.Errorf("GetChunkServerInfo error: %s", err)
		panic(err)
	}

	err = s.GetFidRange(false)
	if err != nil {
		log.Errorf("GetFidRange error: %s", err)
		panic(err)
	}

	go s.GetFidRangeTicker()
	go s.GetChunkServerInfoTicker()

	mysqldriver.InitMeta(s.metadbIp, s.metadbPort, s.metadbUser, s.metadbPassword, s.metaDatabase)
	s.metaDriver = new(mysqldriver.MysqlDriver)

	http.Handle("/", s.router)
	log.Infof("listen: %v", s.Port)
	return http.ListenAndServe(s.Ip + ":" + strconv.Itoa(s.Port), nil)
}
