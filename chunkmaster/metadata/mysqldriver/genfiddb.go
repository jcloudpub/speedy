package mysqldriver

import (
	"fmt"
)

const (
	UPDATE_FID_SQL = "UPDATE gen_fid SET fid = ? "
	GET_FID_SQL    = "SELECT fid FROM gen_fid"
)

func (conn *MySqlConn) UpdateFid(fid uint64) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(UPDATE_FID_SQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(fid)
	if err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) GetFid() (uint64, error) {
	if conn == nil {
		return 0, fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(GET_FID_SQL)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var fid uint64

		err = rows.Scan(&fid)
		if err != nil {
			return 0, err
		}

		return fid, nil
	}

	return 0, fmt.Errorf("fid is empty")
}
