package redisdriver

import (
	. "github.com/jcloudpub/speedy/imageserver/meta"
	"github.com/garyburd/redigo/redis"
	"time"
	"fmt"
	"github.com/jcloudpub/speedy/chunkmaster/util/log"
	"strings"
)

var redisPool = newPool("127.0.0.1:6379")

/*
func (metaInfo *MetaInfo) AddMetaInfoValue(value *MetaInfoValue) error {
	return nil
}*/

type RedisDriver struct{

}

func (r *RedisDriver)StoreMetaInfo(metaInfo *MetaInfo) error {
	if metaInfo.Value.IsLast && metaInfo.Value.Index == 0 {
		err := r.DeleteMetaInfo(metaInfo.Path)
		if err != nil {
			return err
		}
	}

	conn := redisPool.Get()
	defer conn.Close()

	err := r.HandlePath(metaInfo.Path)
	if err != nil {
		return err
	}

	json, err := EncodeJson(metaInfo.Value)
	if err != nil {
		return err
	}

	_, err = conn.Do("LPUSH", metaInfo.Path, json)
	if err != nil {
		return err
	}
	return nil
}

//repositories/username/ubuntu/tag_v2
func (r *RedisDriver)HandlePath(path string) error {
	lastSplitIndex := strings.LastIndex(path, SPLIT)

	if lastSplitIndex == -1 {
		return nil
	}

	lastFragment := path[lastSplitIndex:]

	tagIndex := strings.LastIndex(lastFragment, TAG)

	if tagIndex == -1 {
		return nil
	}

	key := DIRECTORY + path[0:lastSplitIndex]
	value := lastFragment[tagIndex:]

	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("LPUSH", key, value)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisDriver)DeleteMetaInfo(path string) error {
	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", path)

	if err != nil {
		return err
	}

	return nil
}

func (r *RedisDriver)GetDirectoryInfo(path string) ([]string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	interDirectory := DIRECTORY + path

	list, err := conn.Do("LRANGE", interDirectory, 0, -1)

	if err != nil {
		return nil, err
	}

	result := make([]string, 0)

	for _, bts := range list.([]interface{}) {
		result = append(result, string(bts.([]byte)))
	}

	if len(result) == 0 {
		log.Infof("can not find directory info for: %s", interDirectory)
		return nil, fmt.Errorf("can not find directory info for: %s", interDirectory)
	}

	return result, nil
}

func (r *RedisDriver)GetMetaInfoBytes(path string) ([]*MetaInfoValue, error) {
	conn := redisPool.Get()
	defer conn.Close()
	list, err := conn.Do("LRANGE", path, 0, -1)
	if err != nil {
		return nil, err
	}
	metaInfoValues := make([]*MetaInfoValue, 0)
	for _, bts := range list.([]interface {}) {
		jsonMap, err := DecodeJson(bts.([]byte))
		if err != nil {
			return nil, err
		}
		metaInfoValue := new(MetaInfoValue)
		metaInfoValue.Index = uint64(jsonMap["Index"].(float64))
		//metaInfoValue.FileId = uint64(jsonMap["FileId"].(float64))
		//metaInfoValue.GroupId = uint16(jsonMap["GroupId"].(float64))
		metaInfoValue.Start = uint64(jsonMap["Start"].(float64))
		metaInfoValue.End = uint64(jsonMap["End"].(float64))
		metaInfoValue.IsLast = jsonMap["IsLast"].(bool)
		metaInfoValues = append(metaInfoValues, metaInfoValue)
	}

	if len(metaInfoValues) == 0 {
		log.Infof("can not find metainfo for path: %s", path)
		return nil, fmt.Errorf("can not find metainfo for path: %s", path)
	}

	return metaInfoValues, nil
}

func (r *RedisDriver)GetMetaInfo(path string) ([]*MetaInfoValue, error) {
	conn := redisPool.Get()
	defer conn.Close()
	list, err := conn.Do("LRANGE", path, 0, -1)
	if err != nil {
		return nil, err
	}
	metaInfoValues := make([]*MetaInfoValue, 0)
	for _, bts := range list.([]interface {}) {
		jsonMap, err := DecodeJson(bts.([]byte))
		if err != nil {
			return nil, err
		}
		metaInfoValue := new(MetaInfoValue)
		metaInfoValue.Index = uint64(jsonMap["Index"].(float64))
		metaInfoValue.FileId = uint64(jsonMap["FileId"].(float64))
		metaInfoValue.GroupId = uint16(jsonMap["GroupId"].(float64))
		metaInfoValue.Start = uint64(jsonMap["Start"].(float64))
		metaInfoValue.End = uint64(jsonMap["End"].(float64))
		metaInfoValue.IsLast = jsonMap["IsLast"].(bool)
		metaInfoValues = append(metaInfoValues, metaInfoValue)
	}
	return metaInfoValues, nil
}

//func GetMetaInfoValue(path, index, start, end string) (*MetaInfoValue, error) {
func (r *RedisDriver)GetMetaInfoValue(path string, index, start, end uint64) (*MetaInfoValue, error) {
	metaInfoValues, err := r.GetMetaInfo(path)
	if err != nil {
		return nil, err
	}
	var metaInfoValue *MetaInfoValue = nil

	for _, temp := range metaInfoValues {
		if index == temp.Index &&
			start == temp.Start &&
			end == temp.End {
			metaInfoValue = temp
			break
		}
	}

	if metaInfoValue == nil {
		return nil, fmt.Errorf("can not find metainfo")
	}

	return metaInfoValue, nil
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     20,
		IdleTimeout: 60 * time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
