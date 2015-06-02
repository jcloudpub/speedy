package mysqldriver

import (
	"fmt"
	"github.com/jcloudpub/speedy/chunkmaster/metadata"
	"github.com/jcloudpub/speedy/utils"
)

const (
	ADD_CHUNKSERVER_SQL = "INSERT INTO chunkserver (chunkserver_id, group_id, ip, port, status," +
		" total_free_space, max_free_space, pend_writes, writing_count, data_path, " +
		" reading_count, total_chunks, conn_counts, create_time) " +
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())"

	EXIST_CHUNKSERVER_SQL  = "SELECT * FROM chunkserver WHERE group_id = ? AND ip = ? AND port = ? AND deleted = 0"
	UPDATE_CHUNKSERVER_SQL = "UPDATE chunkserver SET total_free_space = ?, max_free_space = ?, pend_writes = ?, writing_count = ?, " +
		" data_path = ?, reading_count = ?, total_chunks = ?, conn_counts = ? " +
		" WHERE group_id = ? AND ip = ? AND port = ? AND deleted = 0"

	UPDATE_CHUNKSERVER_INFO_SQL = "UPDATE chunkserver SET total_free_space = ?, max_free_space = ?, pend_writes = ?, writing_count = ?, " +
		" data_path = ?, reading_count = ?, total_chunks = ?, conn_counts = ? , status = ?" +
		" WHERE group_id = ? AND ip = ? AND port = ? AND status = ? AND deleted = 0"

	UPDATE_CHUNKSERVER_STATUS_SQL = "UPDATE chunkserver SET  status = ?" +
		" WHERE group_id = ? AND ip = ? AND port = ? AND status = ? AND deleted = 0"

	UPDATE_CHUNKSERVER_NORMAL_STATUS = "UPDATE chunkserver SET abnormal_count = 0, status = ? WHERE ip = ? AND port = ? AND deleted = 0 AND status != ?"

	UPDATE_CHUNKSERVER_ERROR_STATUS = "UPDATE chunkserver SET status = ? WHERE ip = ? AND port = ? AND deleted = 0 AND  abnormal_count > ?"

	LIST_CHUNKSERVER_GROUPID_SQL = "SELECT chunkserver_id, group_id, ip, port, " +
		"status, global_status, total_free_space, max_free_space, pend_writes, " +
		"writing_count, data_path, reading_count, total_chunks, conn_counts " +
		" FROM chunkserver WHERE deleted = 0 and group_id=?"

	LIST_CHUNKSERVER_SQL = "SELECT chunkserver_id, group_id, ip, port, " +
		"status, global_status, total_free_space, max_free_space, pend_writes, " +
		"writing_count, data_path, reading_count, total_chunks, conn_counts " +
		" FROM chunkserver WHERE deleted = 0 "
)

func (conn *MySqlConn) AddChunkserver(chunkserver *metadata.Chunkserver) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(ADD_CHUNKSERVER_SQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if _, err := stmt.Exec(util.GenerateRandomID(), chunkserver.GroupId, chunkserver.Ip,
		chunkserver.Port, chunkserver.Status, chunkserver.TotalFreeSpace, chunkserver.MaxFreeSpace,
		chunkserver.PendingWrites, chunkserver.WritingCount, chunkserver.DataDir, chunkserver.ReadingCount,
		chunkserver.TotalChunks, chunkserver.ConnectionsCount); err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) IsExistChunkserver(chunkserver *metadata.Chunkserver) (bool, error) {
	exist := false

	if conn == nil {
		return false, fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(EXIST_CHUNKSERVER_SQL)
	if err != nil {
		return exist, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(chunkserver.GroupId, chunkserver.Ip, chunkserver.Port)
	if err != nil {
		return exist, err
	}
	defer rows.Close()

	if rows.Next() {
		exist = true
	}

	return exist, nil
}

func (conn *MySqlConn) UpdateChunkserverInfo(chunkserver *metadata.Chunkserver, preStatus int, status int) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}
	stmt, err := conn.db.Prepare(UPDATE_CHUNKSERVER_INFO_SQL)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(chunkserver.TotalFreeSpace, chunkserver.MaxFreeSpace, chunkserver.PendingWrites,
		chunkserver.WritingCount, chunkserver.DataDir, chunkserver.ReadingCount, chunkserver.TotalChunks,
		chunkserver.ConnectionsCount, status, chunkserver.GroupId, chunkserver.Ip, chunkserver.Port, preStatus)

	if err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) UpdateChunkserverStatus(chunkserver *metadata.Chunkserver, preStatus int, status int) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}
	stmt, err := conn.db.Prepare(UPDATE_CHUNKSERVER_STATUS_SQL)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(status, chunkserver.GroupId, chunkserver.Ip, chunkserver.Port, preStatus)

	if err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) UpdateChunkserverNORMAL(ip string, port, status, errStatus int) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(UPDATE_CHUNKSERVER_NORMAL_STATUS)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, ip, port, errStatus)
	if err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) UpdateChunkserverERROR(ip string, port, status, count int) error {
	if conn == nil {
		return fmt.Errorf("MySqlConn is nil")
	}

	stmt, err := conn.db.Prepare(UPDATE_CHUNKSERVER_ERROR_STATUS)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, ip, port, count)
	if err != nil {
		return err
	}

	return nil
}

func (conn *MySqlConn) ListChunkserver() (metadata.Chunkservers, error) {
	if conn == nil {
		return nil, fmt.Errorf("MySqlConn is nil")
	}

	chunkservers := make(metadata.Chunkservers, 0, 10)

	stmt, err := conn.db.Prepare(LIST_CHUNKSERVER_SQL)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		chunkserver := new(metadata.Chunkserver)
		err = rows.Scan(&chunkserver.Id, &chunkserver.GroupId, &chunkserver.Ip, &chunkserver.Port,
			&chunkserver.Status, &chunkserver.GlobalStatus, &chunkserver.TotalFreeSpace, &chunkserver.MaxFreeSpace,
			&chunkserver.PendingWrites, &chunkserver.WritingCount, &chunkserver.DataDir, &chunkserver.ReadingCount,
			&chunkserver.TotalChunks, &chunkserver.ConnectionsCount)
		if err != nil {
			return nil, err
		}
		chunkservers = append(chunkservers, chunkserver)
	}

	return chunkservers, nil
}

func (conn *MySqlConn) ListChunkserverGroup(groupId int) (metadata.Chunkservers, error) {
	if conn == nil {
		return nil, fmt.Errorf("MySqlConn is nil")
	}

	chunkservers := make(metadata.Chunkservers, 0, 3)

	stmt, err := conn.db.Prepare(LIST_CHUNKSERVER_GROUPID_SQL)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		chunkserver := new(metadata.Chunkserver)
		err = rows.Scan(&chunkserver.Id, &chunkserver.GroupId, &chunkserver.Ip, &chunkserver.Port,
			&chunkserver.Status, &chunkserver.GlobalStatus, &chunkserver.TotalFreeSpace, &chunkserver.MaxFreeSpace,
			&chunkserver.PendingWrites, &chunkserver.WritingCount, &chunkserver.DataDir, &chunkserver.ReadingCount,
			&chunkserver.TotalChunks, &chunkserver.ConnectionsCount)
		if err != nil {
			return nil, err
		}
		chunkservers = append(chunkservers, chunkserver)
	}
	return chunkservers, nil
}
