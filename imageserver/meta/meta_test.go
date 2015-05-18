package meta

/*
import (
	"testing"
	"fmt"
)

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
	err := StoreMetaInfo(metadata)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMetaInfo(t *testing.T) {
	metaInfoValues, err := GetMetaInfo("first4")
	if err != nil {
		t.Fatal(err)
	}

	for _, metaInfoValue := range metaInfoValues {
		fmt.Println(metaInfoValue)
	}
}

func TestGetMetaInfoBytes(t *testing.T) {
	metaInfoValueBytes, err := GetMetaInfoBytes("first6")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(metaInfoValueBytes))
}

func TestGetMetaInfoValue(t *testing.T) {
	metaInfoValue, err := GetMetaInfoValue("first4", 1, 1, 4)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(metaInfoValue)
}
*/
