package meta

import (
	"time"
)

const (
	TAG       = "tag_"
	SPLIT     = "/"
	DIRECTORY = "DIRECTORY_"
)

//TODO
//groupId == 0 or fileId == 0
//may cause error
type MetaInfoValue struct {
	Index   uint64
	Start   uint64
	End     uint64
	GroupId uint16 `json:",omitempty"`
	FileId  uint64 `json:",omitempty"`
	IsLast  bool
	ModTime time.Time `json:",omitempty`
}

type MetaInfo struct {
	Path  string
	Value *MetaInfoValue
}

type MetaDriver interface {
	StoreMetaInfoV1(metaInfo *MetaInfo) error
	StoreMetaInfoV2(metaInfo *MetaInfo) error
	DeleteFileMetaInfoV1(path string) error
	DeleteFileMetaInfoV2(path string) error
	GetDirectoryInfo(path string) ([]string, error)
	GetDescendantPath(path string) ([]string, error)
	MoveFile(sourcePath, destPath string) error
	GetFileMetaInfo(path string, detail bool) ([]*MetaInfoValue, error)
	GetFragmentMetaInfo(path string, index, start, end uint64) (*MetaInfoValue, error)
}
