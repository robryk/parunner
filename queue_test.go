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
	buf := mq.Shutdown()
	if len(buf) > 0 {
		t.Errorf("Expected no remaining messages, got %d: %v", len(buf), buf)
	}
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
	buf := mq.Shutdown()
	if len(buf) > 0 {
		t.Errorf("Expected no remaining messages, got %d: %v", len(buf), buf)
	}
}

func TestQueueShutdown(t *testing.T) {
	// TODO: figure out what should happen to a get after shutdown
	mq := NewMessageQueue(nil)
	const N = 1000
	putRange(mq, N)
	buf := mq.Shutdown()
	if len(buf) != N {
		t.Fatalf("Found %d messages in queue, wanted %d.", len(buf), N)
	}
	for i, v := range buf {
		if i != v {
			t.Errorf("Expected %d, got %d in queue", i, v)
		}
	}
}
