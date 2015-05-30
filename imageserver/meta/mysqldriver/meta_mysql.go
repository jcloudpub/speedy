package mysqldriver

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/jcloudpub/speedy/imageserver/meta"
	"github.com/jcloudpub/speedy/logs"
	"strings"
	"time"
)

const (
	ADDFILE = 1
	DELFILE = 2
)

type MysqlDriver struct {
}

var mysqlDB *sql.DB

func InitMeta(metadbIp string, metadbPort int, metadbUser, metadbPassword, metaDatabase string) error {
	db, err := newMySqlConn(metadbIp, metadbPort, metadbUser, metadbPassword, metaDatabase)
	if err != nil {
		mysqlDB = nil
		return err
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(20)
	mysqlDB = db

	err = checkConn(mysqlDB)
	if err != nil {
		return err
	}

	go connHeartBeater(mysqlDB)
	return nil
}

func (db *MysqlDriver) StoreMetaInfo(metaInfo *MetaInfo) error {
	if metaInfo.Value.IsLast && metaInfo.Value.Index == 0 {
		err := db.DeleteFileMetaInfo(metaInfo.Path)
		if err != nil {
			return err
		}
	}

	err := db.HandleDirectory(metaInfo.Path, ADDFILE)
	if err != nil {
		return err
	}

	metaInfoValueJson, err := json.Marshal(metaInfo.Value)
	if err != nil {
		return err
	}

	err = pushList(mysqlDB, metaInfo.Path, string(metaInfoValueJson))
	if err != nil {
		return err
	}

	return nil
}

//path: repositories/username/ubuntu/tag_v2
//key: DIRECTORY_repositories/username/ubuntu
//value: tag_v2
func (db *MysqlDriver) ExtractDirectoryAndFile(path string) (string, string) {
	lastSplitIndex := strings.LastIndex(path, SPLIT)
	if lastSplitIndex == -1 {
		return "", ""
	}

	lastFragment := path[lastSplitIndex:]
	tagIndex := strings.Index(lastFragment, TAG)
	if tagIndex != 1 {
		return "", ""
	}

	key := DIRECTORY + path[0:lastSplitIndex]
	value := lastFragment[tagIndex:]

	return key, value
}

//repositories/username/ubuntu/tag_v2
func (db *MysqlDriver) HandleDirectory(path string, opcode int16) error {
	directory, file := db.ExtractDirectoryAndFile(path)
	if len(directory) == 0 || len(file) == 0 {
		return nil
	}

	if opcode == ADDFILE {
		err := pushList(mysqlDB, directory, file)
		if err != nil {
			return err
		}
	} else if opcode == DELFILE {
		err := delListOne(mysqlDB, directory, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *MysqlDriver) DeleteFileMetaInfo(path string) error {
	err := delList(mysqlDB, path)
	if err != nil {
		return err
	}

	err = db.HandleDirectory(path, DELFILE)
	if err != nil {
		return err
	}

	return nil
}

func (db *MysqlDriver) GetDirectoryInfo(path string) ([]string, error) {
	interDirectory := DIRECTORY + path

	files, err := getList(mysqlDB, interDirectory)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		log.Infof("[MysqlDriver] can not find directory info for: %s", interDirectory)
	}

	return files, nil
}

func (db *MysqlDriver) GetFileMetaInfo(path string, detail bool) ([]*MetaInfoValue, error) {
	list, err := getList(mysqlDB, path)
	if err != nil {
		return nil, err
	}

	metaInfoValues := make([]*MetaInfoValue, 0)

	for _, bts := range list {
		var jsonMap map[string]interface{}
		err := json.Unmarshal([]byte(bts), &jsonMap)
		if err != nil {
			return nil, err
		}

		metaInfoValue := new(MetaInfoValue)
		metaInfoValue.Index = uint64(jsonMap["Index"].(float64))
		metaInfoValue.Start = uint64(jsonMap["Start"].(float64))
		metaInfoValue.End = uint64(jsonMap["End"].(float64))
		metaInfoValue.IsLast = jsonMap["IsLast"].(bool)

		if detail {
			metaInfoValue.FileId = uint64(jsonMap["FileId"].(float64))
			metaInfoValue.GroupId = uint16(jsonMap["GroupId"].(float64))
		}

		metaInfoValues = append(metaInfoValues, metaInfoValue)
	}

	return metaInfoValues, nil
}

func (db *MysqlDriver) GetFragmentMetaInfo(path string, index, start, end uint64) (*MetaInfoValue, error) {
	metaInfoValues, err := db.GetFileMetaInfo(path, true)
	if err != nil {
		return nil, err
	}

	var metaInfoValue *MetaInfoValue = nil
	for _, temp := range metaInfoValues {
		if index == temp.Index && start == temp.Start && end == temp.End {
			metaInfoValue = temp
			break
		}
	}

	if metaInfoValue == nil {
		log.Infof("can not find fragment metainfo, path: %s, index: %d, start: %d, end: %d", path, index, start, end)
	}

	return metaInfoValue, nil
}

func newMySqlConn(ip string, port int, user string, passwd string, database string) (*sql.DB, error) {
	args := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", user, passwd, ip, port, database)
	db, err := sql.Open("mysql", args)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func connHeartBeater(conn *sql.DB) {
	for {
		time.Sleep(10 * time.Second)

		err := checkConn(conn)
		if err != nil {
			log.Infof("[connHeartBeater] error: %s", err.Error())
		} else {
			log.Debugf("mysql connHeartBeater OK")
		}
	}
}

func pushList(db *sql.DB, key, value string) error {
	stmt, err := db.Prepare("INSERT INTO key_list (list_key, list_value, md5_key, create_time) VALUES (?, ?, ?, now())")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, value, encrypt([]byte(key)))
	if err != nil {
		return err
	}

	return nil
}

func delList(db *sql.DB, key string) error {
	stmt, err := db.Prepare("DELETE FROM key_list WHERE md5_key = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(encrypt([]byte(key)))
	if err != nil {
		return err
	}

	return nil
}

func delListOne(db *sql.DB, key, value string) error {
	stmt, err := db.Prepare("DELETE FROM key_list WHERE md5_key = ? and list_value = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(encrypt([]byte(key)), value)
	if err != nil {
		return err
	}

	return nil
}

func getList(db *sql.DB, key string) ([]string, error) {
	stmt, err := db.Prepare("SELECT list_value FROM key_list WHERE md5_key = ? ORDER BY create_time DESC")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(encrypt([]byte(key)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values = make([]string, 0)

	for rows.Next() {
		var value string
		err = rows.Scan(&value)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, nil
}

func checkConn(db *sql.DB) error {
	stmt, err := db.Prepare("SELECT count(0) from key_list")
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var count int64
		err := rows.Scan(&count)
		if err != nil {
			return err
		}
	}

	return nil
}

func encrypt(source []byte) string {
	result := md5.Sum(source)
	return fmt.Sprintf("%x", result)
}
