package chunkserver

import (
	"fmt"
	"github.com/jcloudpub/speedy/imageserver/util/log"
	"sync"
	"time"
)

type ChunkServerConnectionPool struct {
	mu    sync.Mutex
	Pools map[string]*ConnectionPool // <ip:port>:connectionpool
}

func NewChunkServerConnectionPool() *ChunkServerConnectionPool {
	return &ChunkServerConnectionPool{
		mu:    sync.Mutex{},
		Pools: make(map[string]*ConnectionPool),
	}
}

func (cscp *ChunkServerConnectionPool) GetConn(chunkserver *ChunkServer) (PoolConnection, error) {
	cscp.mu.Lock()
	defer cscp.mu.Unlock()

	key := fmt.Sprintf("%s:%d", chunkserver.Ip, chunkserver.Port)
	pool, ok := cscp.Pools[key]
	if !ok {
		return nil, fmt.Errorf("pool %s not exist", key)
	}

	return pool.Get()
}

//chunkserver closed, the state of connection in pool is close_wait, need to close those connection
func (cscp *ChunkServerConnectionPool) CheckConnPool(chunkserver *ChunkServer) error {
	for {
		conn, err := cscp.GetConn(chunkserver)
		if err != nil {
			return err
		}

		err = chunkserver.Ping(conn.(*PooledConn))
		if err != nil {
			conn.Close()
			cscp.ReleaseConn(conn)
			continue
		}

		return nil
	}
}

func (cscp *ChunkServerConnectionPool) ReleaseConn(pc PoolConnection) {
	pc.Recycle()
}

func (cscp *ChunkServerConnectionPool) AddPool(chunkserver *ChunkServer) error {
	cscp.mu.Lock()
	defer cscp.mu.Unlock()

	key := fmt.Sprintf("%s:%d", chunkserver.Ip, chunkserver.Port)
	log.Debugf("ChunkServerConnectionPool AddPool: %s", key)

	pool, ok := cscp.Pools[key]
	if ok {
		log.Debugf("AddPool, key: %s already exist", key)
		return nil
	}

	pool = NewConnectionPool("chunk server connection pool", 200, 3600*time.Second)

	log.Debugf("ChunkServerConnectionPool try to open ")
	pool.Open(ConnectionCreator(key))
	log.Debugf("ChunkServerConnectionPool open success")

	cscp.Pools[key] = pool
	return nil
}

func (cscp *ChunkServerConnectionPool) AddExistPool(key string, pool *ConnectionPool) {
	cscp.mu.Lock()
	defer cscp.mu.Unlock()

	log.Debugf("AddExistPool, key: %v, pool: %v", key, pool)

	_, ok := cscp.Pools[key]
	if ok {
		log.Infof("AddExistPool key: %s already exist", key)
		return
	}

	cscp.Pools[key] = pool
	log.Debugf("AddExistPool, key: %v, pool: %v", key, pool)
	return
}

func (cscp *ChunkServerConnectionPool) RemovePool(chunkserver *ChunkServer) {
	cscp.mu.Lock()
	defer cscp.mu.Unlock()

	key := fmt.Sprintf("%s:%d", chunkserver.Ip, chunkserver.Port)
	log.Debugf("RemovePool, key: %v", key)

	delete(cscp.Pools, key)
}

func (cscp *ChunkServerConnectionPool) RemoveAndClosePool(chunkserver *ChunkServer) error {
	cscp.mu.Lock()

	key := fmt.Sprintf("%s:%d", chunkserver.Ip, chunkserver.Port)
	pool, ok := cscp.Pools[key]
	if !ok {
		cscp.mu.Unlock()
		return fmt.Errorf("pool %s not exist", key)
	}

	delete(cscp.Pools, key)

	cscp.mu.Unlock()

	pool.Close() //TODO async close
	return nil
}
