package main

import (
	"io"
	"io/ioutil"
	"os"
	"sync"
)

type FilePipe struct {
	f       *os.File
	readers sync.WaitGroup

	mu      sync.Mutex
	cond    sync.Cond
	size    int64
	closing bool
}

func NewFilePipe() (*FilePipe, error) {
	f, err := ioutil.TempFile("", "filepipe")
	if err != nil {
		return nil, err
	}
	fp := &FilePipe{f: f}
	fp.cond.L = &fp.mu
	return fp, nil
}

func (fp *FilePipe) Reader() io.ReadCloser {
	fp.readers.Add(1)
	return &filePipeReader{fp: fp, pos: 0}
}

func (fp *FilePipe) Write(buf []byte) (int, error) {
	n, err := fp.f.Write(buf)
	fp.mu.Lock()
	fp.size += int64(n)
	fp.cond.Broadcast()
	fp.mu.Unlock()
	return n, err
}

func (fp *FilePipe) Close() error {
	fp.mu.Lock()
	fp.closing = true
	fp.cond.Broadcast()
	fp.mu.Unlock()
	go func() {
		fp.readers.Wait()
		fp.f.Close()
	}()
	return nil
}

type filePipeReader struct {
	fp  *FilePipe
	pos int64
}

func (fpr *filePipeReader) Read(buf []byte) (int, error) {
	for {
		n, err := fpr.fp.f.ReadAt(buf, fpr.pos)
		fpr.pos += int64(n)
		if err == io.EOF {
			err = nil
		}
		if err != nil || n > 0 {
			return n, err
		}
		fpr.fp.mu.Lock()
		for fpr.pos >= fpr.fp.size && !fpr.fp.closing {
			fpr.fp.cond.Wait()
		}
		closing := fpr.fp.closing
		fpr.fp.mu.Unlock()
		if closing {
			return 0, io.EOF
		}
	}
}

func (fpr *filePipeReader) Close() error {
	fpr.fp.readers.Add(-1)
	return nil
}
