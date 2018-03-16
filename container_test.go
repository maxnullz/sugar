package ngxnet

import (
	"testing"
	"time"
)

func Test_MinHeap(t *testing.T) {
	mh := NewMinHeap()
	for i := 1000000; i > 0; i-- {
		mh.Push(i, i)
	}
	tmBefore := time.Now().Nanosecond()
	mh.Update(1000000, 0)
	tmAfter := time.Now().Nanosecond()
	Printf("update one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	tmBefore = tmAfter

	m, p := mh.GetMin()
	Printf("first get min : %v %v \n", m, p)

	for i := 0; i < 10; i++ {
		x := mh.Pop()
		Printf("pop %v  ", x)
	}
	Println("")

	tmBefore = time.Now().Nanosecond()
	mh.Push(0, 10000001)
	tmAfter = time.Now().Nanosecond()
	Printf("push one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)

	tmBefore = tmAfter
	mh.Push(0, 333)
	tmAfter = time.Now().Nanosecond()
	Printf("push one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)

	tmBefore = tmAfter
	i := mh.Pop()
	tmAfter = time.Now().Nanosecond()
	Printf("pop one:%v in heap(len:%v) elapsed %v\n", i, mh.Len(), tmAfter-tmBefore)
	//Println(i, mh.Len())

	tmBefore = tmAfter
	mh.Push(0, 19384)
	i = mh.Pop()
	tmAfter = time.Now().Nanosecond()
	Printf("push and pop one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	//Println(i, mh.Len())
}

func Test_MaxHeap(t *testing.T) {
	mh := NewMaxHeap()
	for i := 1000000; i > 0; i-- {
		mh.Push(0, i)
	}
	m, p := mh.GetMin()

	Printf("max : %v %v \n", m, p)
	for i := 0; i < 10; i++ {
		x := mh.Pop()
		Printf("%v  ", x)
	}

	Println("")

	tmBefore := time.Now().Nanosecond()
	mh.Push(0, 554654)
	tmAfter := time.Now().Nanosecond()
	Printf("push one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	tmBefore = tmAfter
	mh.Push(0, 333)
	tmAfter = time.Now().Nanosecond()
	Printf("push one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	tmBefore = tmAfter
	i := mh.Pop()
	tmAfter = time.Now().Nanosecond()
	Printf("pop one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	Println(i, mh.Len())

	tmBefore = tmAfter
	mh.Push(0, 19384)
	i = mh.Pop()
	tmAfter = time.Now().Nanosecond()
	Printf("push and pop one in heap(len:%v) elapsed %v\n", mh.Len(), tmAfter-tmBefore)
	Println(i, mh.Len())
}
