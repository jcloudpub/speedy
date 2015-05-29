package mysqldriver

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type MySqlConn struct {
	db *sql.DB
}

func NewMySqlConn(ip string, port string, user string, passwd string, database string) (*MySqlConn, error) {
	args := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", user, passwd, ip, port, database)
	db, err := sql.Open("mysql", args)
	if err != nil {
		return nil, err
	}

	return &MySqlConn{db}, nil
}

func (conn *MySqlConn) Close() error {
	return conn.db.Close()
}

func (conn *MySqlConn) SetMaxIdleConns(n int) {
	conn.db.SetMaxIdleConns(n)
}

func (conn *MySqlConn) SetMaxOpenConns(n int) {
	conn.db.SetMaxOpenConns(n)
}
