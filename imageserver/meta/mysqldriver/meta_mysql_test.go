package mysqldriver
/*
import (
	"testing"
	"fmt"
	. "github.com/jcloudpub/speedy/imageserver/meta"
	_"github.com/jcloudpub/speedy/imageserver/meta/redisdriver"
	"sync"
	"time"
)
var metaDriver MetaDriver

func init() {
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

func TestStoreMetaInfoPrefomance(t *testing.T) {
	var wg sync.WaitGroup
	thdCount := 200
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
				//t.Log(metaInfoValue.End)
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
	t.Log("threads ", thdCount, ", exec ", execCount, " total insert req ", thdCount * execCount)
	t.Log("insert per seconds. ", ips)
}

func TestGetMetaInfo(t *testing.T) {
	metaInfoValues, err := metaDriver.GetMetaInfo("first6")
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfoValue := range metaInfoValues {
		fmt.Println(metaInfoValue)
	}
}

func TestGetMetaInfoPerformance(t *testing.T) {
	var wg sync.WaitGroup
	thdCount := 50
	execCount := 6000
	wg.Add(thdCount)

	begin := time.Now()
	for i := 1; i <= thdCount; i++ {
		go func(b int) {
			for j := 1; j <= execCount; j++ {
				_, err := metaDriver.GetMetaInfo(fmt.Sprintf("test%v-%v", b, j))
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
	qps := thdCount * execCount / (int(stime.Seconds()))
	t.Log("threads ", thdCount, ", exec ", execCount, " total select req ", thdCount * execCount)
	t.Log("select per seconds. ", qps)
}


func TestMetaInfoStoreAndGetPrefomance(t *testing.T) {
	var wg sync.WaitGroup
	thdCount := 50
	execCount := 6000
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
				//t.Log(metaInfoValue.End)
				err := metaDriver.StoreMetaInfo(metadata)
				if err != nil {
					t.Fatal(err)
				}

				_, err = metaDriver.GetMetaInfoValue(fmt.Sprintf("test%v-%v", b, j), 1, 1, 29911)
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

	tps := thdCount * execCount / (int(stime.Seconds()))
	t.Log("threads ", thdCount, ", exec ", execCount, " total insert select req ", thdCount * execCount * 2)
	t.Log("tps per seconds. ", tps)
}

func TestGetMetaInfoBytes(t *testing.T) {
	metaInfoValues, err := metaDriver.GetMetaInfoBytes("first6")
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfo := range metaInfoValues {
		t.Log(metaInfo)
	}

}

func TestGetMetaInfoValue(t *testing.T) {
	metaInfoValue, err := metaDriver.GetMetaInfoValue("first6", 1, 1, 29911)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(metaInfoValue)
}
*/
