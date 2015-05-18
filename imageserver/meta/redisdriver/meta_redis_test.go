package redisdriver
/*
import (
	. "github.com/jcloudpub/speedy/imageserver/meta"
	"testing"
	"fmt"
	"time"
	"sync"
)

var metaDriver MetaDriver

func init() {
	metaDriver = new(RedisDriver)
}

func TestStoreMetaInfo(t *testing.T) {
	metadata := new(MetaInfo)
	metaInfoValue := new(MetaInfoValue)
	metaInfoValue.Start = 1
	metaInfoValue.End = 29911
	metaInfoValue.Index = 1
	metaInfoValue.GroupId = 123
	metaInfoValue.FileId = 12312
	metaInfoValue.IsLast = true

	metadata.Path = "first8"
	metadata.Value = metaInfoValue
	t.Log(metaInfoValue.End)
	err := metaDriver.StoreMetaInfo(metadata)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreMetaInfoPrefomance(t *testing.T) {
	var wg sync.WaitGroup
	thdCount := 4000
	execCount := 4000
	wg.Add(thdCount)

	begin := time.Now()
	for i := 1; i <= thdCount; i++ {
		go func() {
			for j := 1; j <= execCount; j++ {
				metadata := new(MetaInfo)
				metaInfoValue := new(MetaInfoValue)
				metaInfoValue.Start = 1
				metaInfoValue.End = 29911
				metaInfoValue.Index = 1
				metaInfoValue.GroupId = 123
				metaInfoValue.FileId = 12312
				metaInfoValue.IsLast = true

				metadata.Path = "first8"
				metadata.Value = metaInfoValue
				//t.Log(metaInfoValue.End)
				err := metaDriver.StoreMetaInfo(metadata)
				if err != nil {
					t.Fatal(err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	end := time.Now()
	t.Log(end.Sub(begin))
}

func TestGetMetaInfo(t *testing.T) {
	metaInfoValues, err := metaDriver.GetMetaInfo("first8")
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfoValue := range metaInfoValues {
		fmt.Println(metaInfoValue)
	}
}

func TestGetMetaInfoBytes(t *testing.T) {
	metaInfoValues, err := metaDriver.GetMetaInfoBytes("first8")
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfo := range metaInfoValues {
		t.Log(metaInfo)
	}

}

func TestGetMetaInfoValue(t *testing.T) {
	metaInfoValue, err := metaDriver.GetMetaInfoValue("first8", 1, 1, 29911)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(metaInfoValue)
}
*/
