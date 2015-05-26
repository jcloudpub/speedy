package chunkserver

import (
	"io"
	"fmt"
	"encoding/binary"
	"bytes"
	"github.com/jcloudpub/speedy/imageserver/meta"
	"github.com/jcloudpub/speedy/imageserver/util/log"
)

const (
	RW_STATUS = 1
	RO_STATUS = 2
	ERR_STATUS = 3

	GLOBAL_NORMAL_STATUS = 0
	GLOBAL_READ_STAUS = 8
)

type ChunkServer struct {
	GroupId int32
	Ip string
	Port int64
	Status int8
	GlobalStatus   int8
	TotalFreeSpace int64
	MaxFreeSpace int64
	PendingWrites	int
	WritingCount   int
}

type ChunkServerGroups struct {
	GroupMap map[string][]ChunkServer //groupId <> []ChunkServer
}

var (
	PUT uint8 = 0x00
	GET uint8 = 0x01
	DELETE uint8 = 0x02
)

func (csgs *ChunkServerGroups) GetChunkServerGroup(groupId string) ([]ChunkServer, bool) {
	group, ok := csgs.GroupMap[groupId]
	return group, ok
}

func (csi *ChunkServer) HostInfoEqual(another *ChunkServer) bool {
	return csi.Ip == another.Ip && csi.Port == another.Port
}

func (cs *ChunkServer) PutData(data []byte, conn *PooledConn, fileId uint64) error {
	output := new(bytes.Buffer)
	header := make([]byte, HEADERSIZE)

	binary.Write(output, binary.BigEndian, PUT)
	binary.Write(output, binary.BigEndian, uint32(len(data) + 2 + 8))
	binary.Write(output, binary.BigEndian, uint16(cs.GroupId))
	binary.Write(output, binary.BigEndian, uint64(fileId))

	log.Debugf("groupId: %d, fileId: %d", cs.GroupId, fileId)

	output.Write(data)
	_, err := conn.Write(output.Bytes())
	if err != nil {
		log.Errorf("write conn error: %s", err)
		return err
	}

	if _, err := io.ReadFull(conn.br, header); err != nil {
		log.Errorf("read header error: %s", err)
		return err
	}

	if header[0] == PUT && header[1] == 0 {
		log.Debugf("upload success")
		return nil
	}

	log.Errorf("fileId: %d, upload failed, header[0] = %d, header[1] = %d", fileId, header[0], header[1])
	return fmt.Errorf("upload error, code: %d", header[1])
}

func (cs *ChunkServer) GetData(miv *meta.MetaInfoValue, conn *PooledConn) ([]byte, error) {
	output := new(bytes.Buffer)
	header := make([]byte, HEADERSIZE)

	binary.Write(output, binary.BigEndian, GET)
	binary.Write(output, binary.BigEndian, uint32(2+8))
	binary.Write(output, binary.BigEndian, uint16(miv.GroupId))
	binary.Write(output, binary.BigEndian, uint64(miv.FileId))

	_, err := conn.Conn.Write(output.Bytes())
	if err != nil {
		fmt.Errorf("write socket error %s\n", err)
		return nil, err
	}

	_, err = io.ReadFull(conn.br, header)
	if err != nil {
		log.Errorf("GetData read header error: %s", err)
		return nil, err
	}

	log.Debugf("%s, download file, header[0] = %d, code = %d", miv, header[0], header[1])
	if header[0] != GET || header[1] != 0 {
		log.Errorf("%s, download file failed, header[0] = %d, code = %d", miv, header[0], header[1])
		return nil, fmt.Errorf("download file failed, code = %d\n", header[1])
	}

	bodyLen := binary.BigEndian.Uint32(header[2:])
	data := make([]byte, bodyLen)
	log.Debugf("GetData len: %d, %d", bodyLen, len(data))

	if _, err := io.ReadFull(conn.br, data); err != nil {
		return nil, fmt.Errorf("read socket error %s", err)
	}

	return data, nil
}

func (cs *ChunkServer) DeleteData(groupId, fileId string, conn *PooledConn) error {
	//TODO send headerInfo
	return nil
}

func parseUint32(data []byte) (uint32, error) {
	buf := bytes.NewBuffer(data)
	var x uint32
	err := binary.Read(buf, binary.BigEndian, &x)
	if err != nil {
		return 0, err
	}

	return x, nil
}

func parseUint8(data []byte)(uint8, error) {
	buf := bytes.NewBuffer(data)
	var x uint8
	err := binary.Read(buf, binary.BigEndian, &x)
	if err != nil {
		return 0, err
	}

	return x, nil
}
