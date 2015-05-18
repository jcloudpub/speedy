package mysqldriver

import (
	"testing"
)

func TestUpdateFid(t *testing.T) {
	conn, err := getConn()
	fid := uint64(1000000)
	err = conn.UpdateFid(fid)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetFid(t *testing.T) {
	conn, err := getConn()
	fid, err := conn.GetFid()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fid)
}
