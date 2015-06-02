package metadata

import (
	"github.com/jcloudpub/speedy/utils"
	"time"
)

type Chunkserver struct {
	Id               string `json:"-"`
	GroupId          uint16
	Ip               string
	Port             int
	Status           int       `json:",omitempty"`
	GlobalStatus     int       `json:",omitempty"`
	TotalFreeSpace   int64     `json:",omitempty"`
	MaxFreeSpace     int64     `json:",omitempty"`
	PendingWrites    int       `json:",omitempty"`
	WritingCount     int       `json:",omitempty"`
	ReadingCount     uint32    `json:",omitempty"`
	TotalChunks      uint32    `json:",omitempty"`
	ConnectionsCount uint32    `json:",omitempty"`
	DataDir          string    `json:",omitempty"`
	UpdateTime       time.Time `json:"-"`
}

type Chunkservers []*Chunkserver

type MetaDataDriver interface {
	Close() error

	AddChunkserver(chunkserver *Chunkserver) error
	UpdateChunkserverStatus(chunkserver *Chunkserver, preStatus int, status int) error
	IsExistChunkserver(chunkServer *Chunkserver) (bool, error)
	UpdateChunkserverInfo(chunkserver *Chunkserver, preStatus int, status int) error
	ListChunkserver() (Chunkservers, error)
	ListChunkserverGroup(groupId int) (Chunkservers, error)
	UpdateChunkserverNORMAL(ip string, port, status, count int) error
	UpdateChunkserverERROR(ip string, port, status, count int) error

	GetFid() (uint64, error)
	UpdateFid(fid uint64) error
}

func GenChunkserver(jsonMap map[string]interface{}) (*Chunkserver, error) {
	chunkserver := new(Chunkserver)

	ip, err := util.CheckMapString(jsonMap, "Ip")
	if err != nil {
		return nil, err
	}
	chunkserver.Ip = ip

	port, err := util.CheckMapInt(jsonMap, "Port")
	if err != nil {
		return nil, err
	}
	chunkserver.Port = port

	/*
		status, err := util.CheckMapInt(jsonMap, "Status")
		if err != nil {
			return nil, err
		}
	*/

	//chunkserver.Status = status

	groupId, err := util.CheckMapUInt16(jsonMap, "GroupId")
	if err != nil {
		return nil, err
	}
	chunkserver.GroupId = groupId

	maxFreeSpace, err := util.CheckMapInt64(jsonMap, "MaxFreeSpace")
	if err != nil {
		return nil, err
	}
	chunkserver.MaxFreeSpace = maxFreeSpace

	totalFreeSpace, err := util.CheckMapInt64(jsonMap, "TotalFreeSpace")
	if err != nil {
		return nil, err
	}
	chunkserver.TotalFreeSpace = totalFreeSpace

	pendingWrites, err := util.CheckMapInt(jsonMap, "PendingWrites")
	if err != nil {
		return nil, err
	}
	chunkserver.PendingWrites = pendingWrites

	writtingCount, err := util.CheckMapInt(jsonMap, "WritingCount")
	if err != nil {
		return nil, err
	}
	chunkserver.WritingCount = writtingCount

	dataDir, err := util.CheckMapString(jsonMap, "DataDir")
	if err != nil {
		return nil, err
	}
	chunkserver.DataDir = dataDir

	readCount, err := util.CheckMapUInt32(jsonMap, "ReadingCount")
	if err != nil {
		return nil, err
	}
	chunkserver.ReadingCount = readCount

	totalChunks, err := util.CheckMapUInt32(jsonMap, "TotalChunks")
	if err != nil {
		return nil, err
	}
	chunkserver.TotalChunks = totalChunks

	connectionsCount, err := util.CheckMapUInt32(jsonMap, "ConnectionsCount")
	if err != nil {
		return nil, err
	}
	chunkserver.ConnectionsCount = connectionsCount

	return chunkserver, nil
}
