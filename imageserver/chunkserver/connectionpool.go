package chunkserver

import (
	"errors"
	"github.com/jcloudpub/speedy/imageserver/pools"
	"github.com/jcloudpub/speedy/imageserver/util/log"
	"sync"
	"time"
)

var (
	CONN_POOL_CLOSED_ERR = errors.New("connection pool is closed")
)

type PoolConnection interface {
	Close()
	IsClosed() bool
	Recycle()
}

// CreateConnectionFunc is the factory method to create new connections
// within the passed ConnectionPool.
type CreateConnectionFunc func(*ConnectionPool) (connection PoolConnection, err error)

type ConnectionPool struct {
	mu          *sync.Mutex
	connections *pools.ResourcePool
	capacity    int
	idleTimeout time.Duration
}

func NewConnectionPool(name string, capacity int, idleTimeout time.Duration) *ConnectionPool {
	cp := &ConnectionPool{
		mu:          &sync.Mutex{},
		capacity:    capacity,
		idleTimeout: idleTimeout,
	}

	if name == "" {
		return cp
	}
	//TODO log
	return cp
}

func (cp *ConnectionPool) pool() (p *pools.ResourcePool) {
	cp.mu.Lock()
	p = cp.connections
	cp.mu.Unlock()
	return p
}

//Open must be cal before starting to use the pool
func (cp *ConnectionPool) Open(connFactory CreateConnectionFunc) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	f := func() (pools.Resource, error) {
		return connFactory(cp)
	}

	log.Debugf("connectionPool open == begin")
	cp.connections = pools.NewResourcePool(f, cp.capacity, cp.capacity, cp.idleTimeout)
	log.Debugf("connectionPool open == end")
}

// Close will close the pool and wait for connections to be returned before
// exiting
func (cp *ConnectionPool) Close() {
	p := cp.pool()
	if p == nil {
		return
	}

	p.Close()
	cp.mu.Lock()
	cp.connections = nil
	cp.mu.Unlock()
}

func (cp *ConnectionPool) Get() (PoolConnection, error) {
	p := cp.pool()
	if p == nil {
		return nil, CONN_POOL_CLOSED_ERR
	}

	r, err := p.Get()
	if err != nil {
		return nil, err
	}

	return r.(PoolConnection), nil
}

func (cp *ConnectionPool) TryGet() (PoolConnection, error) {
	p := cp.pool()

	if p == nil {
		return nil, CONN_POOL_CLOSED_ERR
	}

	r, err := p.TryGet()
	if err != nil || r == nil {
		return nil, err
	}

	return r.(PoolConnection), nil
}

func (cp *ConnectionPool) Put(conn PoolConnection) {
	p := cp.pool()
	if p == nil {
		panic(CONN_POOL_CLOSED_ERR)
	}
	p.Put(conn)
}

func (cp *ConnectionPool) SetCapacity(capacity int) (err error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.connections == nil {
		return CONN_POOL_CLOSED_ERR
	}

	err = cp.connections.SetCapacity(capacity)
	if err != nil {
		return err
	}
	cp.capacity = capacity

	return nil
}

func (cp *ConnectionPool) SetIdleTimeOut(idleTimeout time.Duration) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.connections != nil {
		cp.connections.SetIdleTimeout(idleTimeout)
	}
	cp.idleTimeout = idleTimeout
}

func (cp *ConnectionPool) StatsJSON() string {
	p := cp.pool()
	if p == nil {
		return "{}"
	}
	return p.StatsJSON()
}

func (cp *ConnectionPool) Capacity() int64 {
	p := cp.pool()
	if p == nil {
		return 0
	}
	return p.Capacity()
}

func (cp *ConnectionPool) MaxCap() int64 {
	p := cp.pool()
	if p == nil {
		return 0
	}
	return p.MaxCap()
}

func (cp *ConnectionPool) WaitCount() int64 {
	p := cp.pool()
	if p == nil {
		return 0
	}
	return p.WaitCount()
}

func (cp *ConnectionPool) WaitTime() time.Duration {
	p := cp.pool()
	if p == nil {
		return 0
	}
	return p.WaitTime()
}

func (cp *ConnectionPool) IdleTimeout() time.Duration {
	p := cp.pool()
	if p == nil {
		return 0
	}
	return p.IdleTimeout()
}
