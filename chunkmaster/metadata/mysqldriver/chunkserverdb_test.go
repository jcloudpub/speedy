package mysqldriver

import (
	"github.com/jcloudpub/speedy/chunkmaster/metadata"
	"testing"
)

func getConn() (metadata.MetaDataDriver, error) {
	conn, err := NewMySqlConn("127.0.0.1", "3306", "username", "password", "testdb")
	conn.SetMaxOpenConns(100)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func TestAddChunkserver(t *testing.T) {
	conn, err := getConn()

	chunkserver := new(metadata.Chunkserver)
	chunkserver.GroupId = uint16(65535)
	chunkserver.Ip = "127.0.0.1"
	chunkserver.Port = 5444
	chunkserver.Status = 0
	chunkserver.TotalFreeSpace = int64(45465465)
	chunkserver.MaxFreeSpace = int64(123123123123123)
	chunkserver.PendingWrites = 1
	chunkserver.WritingCount = 1
	chunkserver.DataDir = "/export/"
	chunkserver.ReadingCount = 1
	chunkserver.TotalChunks = 1
	chunkserver.ConnectionsCount = 1

	err = conn.AddChunkserver(chunkserver)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExistChunkserver(t *testing.T) {
	conn, err := getConn()
	chunkserver := new(metadata.Chunkserver)

	chunkserver.GroupId = uint16(1)
	chunkserver.Ip = "127.0.0.1"
	chunkserver.Port = 5444
	chunkserver.Status = 0
	chunkserver.TotalFreeSpace = int64(45465465)
	chunkserver.MaxFreeSpace = int64(123123123123123)

	exist, err := conn.IsExistChunkserver(chunkserver)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("exist:", exist)
}

func TestNotExistChunkserver(t *testing.T) {
	conn, err := getConn()
	chunkserver := new(metadata.Chunkserver)

	chunkserver.GroupId = uint16(3)
	chunkserver.Ip = "127.0.0.1"
	chunkserver.Port = 5444
	chunkserver.Status = 0
	chunkserver.TotalFreeSpace = int64(45465465)
	chunkserver.MaxFreeSpace = int64(123123123123123)

	exist, err := conn.IsExistChunkserver(chunkserver)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("exist:", exist)
}

func TestUpdateChunkserver(t *testing.T) {
	conn, err := getConn()
	chunkserver := new(metadata.Chunkserver)

	chunkserver.GroupId = uint16(1)
	chunkserver.Ip = "127.0.0.1"
	chunkserver.Port = 5444
	chunkserver.Status = 0
	chunkserver.TotalFreeSpace = int64(65)
	chunkserver.MaxFreeSpace = int64(1231231231231234534)

	err = conn.UpdateChunkserver(chunkserver)
	if err != nil {
		t.Fatal(err)
	}
}

func TestListChunkserver(t *testing.T) {
	conn, err := getConn()
	chunkservers, err := conn.ListChunkserver()
	if err != nil {
		t.Fatal(err)
	}

	for _, chunkserver := range chunkservers {
		t.Log("chunkservers: ", chunkserver)
	}
}
