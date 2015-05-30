package chunkserver

import (
	"fmt"
	"github.com/jcloudpub/speedy/logs"
	"time"
)

//[start, end)
type Fids struct {
	Start uint64 `json:"FidBegin"`
	End   uint64 `json:"FidEnd"`
	ch    chan uint64
}

func NewFids() *Fids {
	return &Fids{
		Start: 0,
		End:   0,
		ch:    make(chan uint64, 200),
	}
}

var (
	FIDS_EMPTY_ERR = fmt.Errorf("fids is empty")
)

func (fids *Fids) IsShortage() bool {
	if len(fids.ch) < 20 && len(fids.ch) < cap(fids.ch) {
		return true
	}
	return false
}

func (fids *Fids) ReSet(start, end uint64) {
	for i := start; i < end; i++ {
		if len(fids.ch) < cap(fids.ch) {
			fids.ch <- i
		}
	}
}

func (fids *Fids) Merge(start uint64, end uint64, wait bool) { //[start, end)
	log.Debugf("merge begin, start:%d, end:%d, wait: %s", start, end, wait)

	if !wait {
		for i := start; i < end; i++ {
			select {
			case fids.ch <- i:
				log.Debugf("fid %d put to channel success", i)
			default:
				log.Infof("fid channel is full")
				return
			}
		}
		log.Debugf("merge end == not wait")
		return
	}

	for i := start; i < end; i++ {
		fids.ch <- i
		log.Debugf("%d put to channel success", i)
	}

	log.Debugf("merge end")
}

func (fids *Fids) GetFid() (uint64, error) {
	log.Debugf("GetFid from channel")
	select {
	case fid := <-fids.ch:
		log.Debugf("GetFid success, fid is: %d", fid)
		return fid, nil
	default:
		return 0, FIDS_EMPTY_ERR
	}
}

func (fids *Fids) GetFidWait() (uint64, error) {
	log.Debugf("GetFid from channel")
	select {
	case fid := <-fids.ch:
		log.Debugf("GetFid success, fid is: %d", fid)
		return fid, nil
	case <-time.After(time.Second * 3):
		log.Debugf("GetFid failed, wait timeout")
		return 0, FIDS_EMPTY_ERR
	}
}
