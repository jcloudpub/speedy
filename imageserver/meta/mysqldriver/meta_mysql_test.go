package mysqldriver

import (
	"fmt"
	. "github.com/jcloudpub/speedy/imageserver/meta"
	"sync"
	"testing"
	"time"
)

var metaDriver MetaDriver

func init() {
	InitMeta("127.0.0.1", 3306, "root", "", "metadb")
	metaDriver = new(MysqlDriver)
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

	metadata.Path = "first6"
	metadata.Value = metaInfoValue
	t.Log(metaInfoValue.End)
	err := metaDriver.StoreMetaInfo(metadata)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMetaInfo(t *testing.T) {
	metaInfoValues, err := metaDriver.GetFileMetaInfo("first6", true)
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfoValue := range metaInfoValues {
		fmt.Println(metaInfoValue)
	}
}

func TestStoreMetaInfoPrefomance(t *testing.T) {
	var wg sync.WaitGroup
	thdCount := 1
	execCount := 7500
	wg.Add(thdCount)

	begin := time.Now()
	for i := 1; i <= thdCount; i++ {
		go func(b int) {
			for j := 1; j <= execCount; j++ {
				metadata := new(MetaInfo)
				metaInfoValue := new(MetaInfoValue)
				metaInfoValue.Start = 1
				metaInfoValue.End = 29911
				metaInfoValue.Index = 1
				metaInfoValue.GroupId = 123
				metaInfoValue.FileId = 12312
				metaInfoValue.IsLast = true

				metadata.Path = fmt.Sprintf("test%v-%v", b, j)
				metadata.Value = metaInfoValue
				err := metaDriver.StoreMetaInfo(metadata)
				if err != nil {
					t.Fatal(err)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	end := time.Now()
	stime := end.Sub(begin)
	t.Log(stime)

	ips := thdCount * execCount / (int(stime.Seconds()))
	t.Log("threads ", thdCount, ", exec ", execCount, " total insert req ", thdCount*execCount)
	t.Log("insert per seconds. ", ips)
}

func TestGetFragmentMetaInfoValue(t *testing.T) {
	metaInfoValue, err := metaDriver.GetFragmentMetaInfo("first6", 1, 1, 29911)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(metaInfoValue)
}
