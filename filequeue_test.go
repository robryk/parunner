package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"testing"
)

var testString string

func init() {
	const text = "Mary had a little lamb"
	const N = 100
	var buf bytes.Buffer
	for i := 0; i < N; i++ {
		if _, err := buf.WriteString(text); err != nil {
			panic(err)
		}
	}
	testString = buf.String()
}

func TestFileQueueSimple(t *testing.T) {
	fp, err := NewFilePipe()
	if err != nil {
		t.Fatalf("Failed to create a filepipe: %v", err)
	}
	finished := make(chan struct{})
	fpr := fp.Reader()
	go func() {
		buf, err := ioutil.ReadAll(fpr)
		if err != nil {
			t.Errorf("Failed to read from a filepipe reader: %v", err)
		} else if string(buf) != testString {
			t.Errorf("Read %v, expected %v", string(buf), testString)
		}
		if err := fpr.Close(); err != nil {
			t.Errorf("Failed to close a filepipe reader: %v", err)
		}
		finished <- struct{}{}
	}()
	_, err = fp.Write([]byte(testString))
	if err != nil {
		t.Errorf("Failed to write to a filepipe: %v", err)
	}
	err = fp.Close()
	if err != nil {
		t.Errorf("Failed to close a filepipe: %v", err)
	}
	<-finished
}

func TestFileQueueConcurrentReaders(t *testing.T) {
	fp, err := NewFilePipe()
	if err != nil {
		t.Fatalf("Failed to create a filepipe: %v", err)
	}
	var wg sync.WaitGroup
	const P = 10
	for i := 0; i < P; i++ {
		wg.Add(1)
		fpr := fp.Reader()
		go func(fpr io.ReadCloser) {
			buf, err := ioutil.ReadAll(fpr)
			if err != nil {
				t.Fatalf("Failed to read from a filepipe reader: %v", err)
			}
			if string(buf) != testString {
				t.Errorf("Read %v, expected %v", string(buf), testString)
			}
			if err := fpr.Close(); err != nil {
				t.Errorf("Failed to close a filepipe reader: %v", err)
			}
			wg.Add(-1)
		}(fpr)
	}
	_, err = fp.Write([]byte(testString))
	if err != nil {
		t.Errorf("Failed to write to a filepipe: %v", err)
	}
	err = fp.Close()
	if err != nil {
		t.Errorf("Failed to close a filepipe: %v", err)
	}
	wg.Wait()
}
