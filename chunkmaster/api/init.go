package api

import (
	"github.com/jcloudpub/speedy/chunkmaster/metadata"
	"github.com/jcloudpub/speedy/chunkmaster/metadata/mysqldriver"
	"github.com/jcloudpub/speedy/logs"
	"sync"
)

var mdDriver metadata.MetaDataDriver
var fid *Fid
var lock sync.RWMutex
var serverInfo map[string]*metadata.Chunkserver //key[groupId:ip:port]--chunkserver

type Fid struct {
	sync.Mutex
	Begin uint64
	End   uint64
}

func InitAll(host, port, user, passwd, db string) {
	mdDriver = newMysqlDriver(host, port, user, passwd, db)
	fid = newFid()
}

func newMysqlDriver(host, port, user, passwd, db string) *mysqldriver.MySqlConn {
	conn, err := mysqldriver.NewMySqlConn(host, port, user, passwd, db)
	if err != nil {
		log.Errorf("create MysqlConn faild when init package api")
		return nil
	}
	conn.SetMaxIdleConns(10)
	conn.SetMaxOpenConns(100)
	log.Infof("create mysqlConn ok")
	return conn
}

func newFid() *Fid {
	fid := new(Fid)
	fid.Begin = 0
	fid.End = 0
	return fid
}
