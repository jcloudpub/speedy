package chunkserver

import (
	"testing"
)

func TestMaxHeadp(t *testing.T) {
	minHeap := NewMinHeap(7)

	minHeap.AddElement("c", 10, 0, 9)
	Print(minHeap, t)
	minHeap.AddElement("b", 20, 0, 8)
	Print(minHeap, t)
	minHeap.AddElement("a", 40, 0, 7)
	Print(minHeap, t)
	minHeap.AddElement("f", 40, 1, 10)
	Print(minHeap, t)
	minHeap.AddElement("h", 60, 2, 10)
	Print(minHeap, t)

	minHeap.AddElement("g", 70, 0, 10)
	Print(minHeap, t)

	minHeap.AddElement("i", 80, 0, 4)
	Print(minHeap, t)

	minHeap.AddElement("u", 90, 0, 5)
	Print(minHeap, t)

	minHeap.AddElement("x", 34, 0, 3)
	Print(minHeap, t)

	minHeap.AddElement("q", 56, 2, 10)
	Print(minHeap, t)

	minHeap.AddElement("r", 102, 1, 10)
	Print(minHeap, t)

	minHeap.AddElement("t", 65, 0, 2)
	Print(minHeap, t)

	minHeap.AddElement("z", 103, 0, 3)
	Print(minHeap, t)
	t.Log("+++++++++++++++++++++++++++++++++")

	minHeap.BuildMinHeapSecondary()
	Print(minHeap, t)
	t.Log("+++++++++++++++++++++++++++++++++")
}

func Print(minHeap *MinHeap, t *testing.T) {
	t.Log("=========Print begin===============")
	for index := 0; index < minHeap.size; index++ {
		t.Log(minHeap.arr[index].GroupId, ": %s", minHeap.arr[index])
	}
	t.Log("=========Print end================")
}
