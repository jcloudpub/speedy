package chunkserver

import (
	"fmt"
	"github.com/jcloudpub/speedy/imageserver/util/log"
)

type HeapElement struct {
	GroupId string
	FreeSpace int64
	PendingWrites	int
	WritingCount   int
}

type MinHeap struct {
	capacity int
	size int
	arr []*HeapElement
}

func NewMinHeap(capa int) *MinHeap {
	return &MinHeap{
		capacity	: capa,
		size		: 0,
		arr 		: make([]*HeapElement, capa),
	}
}

func (h *MinHeap) GetElementGroupId(index int) (string, error) {
	if index >= h.size {
		return "", fmt.Errorf("index: %d > h.Size: %d", index, h.size)
	}

	return h.arr[index].GroupId, nil
}

func (h *MinHeap) AddElement(groupId string, freeSpace int64, pendingWrites int, writingCount int) {
	ele := &HeapElement{
		GroupId: groupId,
		FreeSpace: freeSpace,
		PendingWrites: pendingWrites,
		WritingCount: writingCount,
	}

	if h.size < h.capacity {
		h.arr[h.size] = ele
		h.size++

		if h.size == h.capacity {
			log.Debugf("h.size: %d == h.capacity: %d", h.size, h.capacity)
			h.buildMinHeap()
		}

		return
	}

	if ele.FreeSpace > h.arr[0].FreeSpace {
		log.Debugf("ele.FreeSpace: %d > min.FreeSpace: %d", ele.FreeSpace, h.arr[0].FreeSpace)
		h.arr[0] = ele
		h.buildMinHeap()
	}
}

func (h *MinHeap) GetSize() int {
	return h.size
}

func (h *MinHeap) buildMinHeap() {
	log.Debugf("buildMinHeap ==== begin")
	for index := h.size/2 - 1; index >= 0; index-- {
		h.minHeapify(index)
	}

	for index := 0; index < h.size; index++ {
		log.Debugf(h.arr[index].GroupId, ": ", h.arr[index].FreeSpace)
	}

	log.Debugf("buildMinHeap ==== end")
}

func (h *MinHeap) BuildMinHeapSecondary() {
	log.Debugf("rebuild ==== begin")
	for index := h.size/2 - 1; index >= 0; index-- {
		h.MinHeapifySecondary(index)
	}

	for index := 0; index < h.size; index++ {
		log.Debugf("%s", h.arr[index])
	}
	log.Debugf("rebuild ==== end")
}

func (h *MinHeap) minHeapify(index int) {
	leftIndex := left(index)
	rightIndex := right(index)

	smallest := h.compare(index, leftIndex)
	smallest = h.compare(smallest, rightIndex)

	if smallest != index {
		tempElement := 	h.arr[smallest]
		h.arr[smallest] = h.arr[index]
		h.arr[index] = tempElement
		h.minHeapify(smallest)
	}
}

func (h *MinHeap) MinHeapifySecondary(index int) {
	leftIndex := left(index)
	rightIndex := right(index)

	smallest := h.compareSecondary(index, leftIndex)
	smallest = h.compareSecondary(smallest, rightIndex)

	if smallest != index {
		tempElement := 	h.arr[smallest]
		h.arr[smallest] = h.arr[index]
		h.arr[index] = tempElement
		h.MinHeapifySecondary(smallest)
	}
}

//the samller is freespace is smaller
func (h *MinHeap) compare(parent int, child int) int {
	length := h.size

	if (child < length) && (h.arr[parent].FreeSpace > h.arr[child].FreeSpace) {
		return child
	}

	return parent
}

//the smaller is writingcount is smaller or pendingwrites is smaller
func (h *MinHeap) compareSecondary(parent int, child int) int {
	length := h.size

	if (child < length) {
		if h.arr[parent].PendingWrites == 0 && h.arr[child].PendingWrites == 0 {
			if h.arr[parent].WritingCount > h.arr[child].WritingCount {
				return child
			}
			return parent
		}

		if h.arr[parent].PendingWrites > h.arr[child].PendingWrites {
			return child
		}
	}

	return parent
}

func left(index int) int{
	return index*2 + 1
}

func right(index int) int{
	return index*2 + 2
}
