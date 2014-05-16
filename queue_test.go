package main

import (
	"testing"
	"time"
)

func putRange(mq *MessageQueue, n int) {
	for i := 0; i < n; i++ {
		mq.Put(i)
	}
}

func TestQueueSimple(t *testing.T) {
	mq := NewMessageQueue(nil)
	const N = 1000
	go putRange(mq, N)
	for i := 0; i < N; i++ {
		if got := mq.Get(); got != i {
			t.Errorf("When reading from a queue got=%v want=%v", got, i)
		}
	}
	mq.Shutdown()
}

func TestQueueSelect(t *testing.T) {
	selector := make(chan *MessageQueue)
	mq := NewMessageQueue(selector)
	const N = 1000
	select {
	case _ = <-selector:
		t.Error("Selecter strobing spuriously")
	case _ = <-time.After(50 * time.Millisecond):
	}
	go putRange(mq, N)
	for i := 0; i < N; i++ {
		_ = <-selector
		if got := mq.Get(); got != i {
			t.Errorf("When reading from a queue got=%v want=%v", got, i)
		}
	}
	mq.Shutdown()
}
