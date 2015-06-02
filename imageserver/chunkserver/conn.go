package chunkserver

import (
	"bufio"
	"net"
)

type Conn struct {
	addr string
	net.Conn
	closed bool
	br     *bufio.Reader
}

func (c *Conn) Close() {
	c.Conn.Close()
	c.closed = true
}

func (c *Conn) IsClosed() bool {
	return c.closed
}

type PooledConn struct {
	*Conn
	pool *ConnectionPool
}

func (pc *PooledConn) Recycle() {
	if pc.IsClosed() {
		pc.pool.Put(nil)
	} else {
		pc.pool.Put(pc)
	}
}

func NewConnection(addr string) (*Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Conn{
		addr: addr,
		Conn: conn,
		br:   bufio.NewReaderSize(conn, 1024*1024*1),
	}, nil
}

func ConnectionCreator(addr string) CreateConnectionFunc {
	return func(pool *ConnectionPool) (PoolConnection, error) {
		c, err := NewConnection(addr)
		if err != nil {
			return nil, err
		}

		return &PooledConn{c, pool}, nil
	}
}
