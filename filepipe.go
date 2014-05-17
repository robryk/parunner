package main

import (
	"io"
	"io/ioutil"
	"os"
	"sync"
)

// FilePipe is a tailable buffer backed by a file.
type FilePipe struct {
	f *os.File

	mu      sync.Mutex
	cond    sync.Cond
	size    int64
	closing bool
}

// Create a new FilePipe backed by a temporary file.
func NewFilePipe() (*FilePipe, error) {
	f, err := ioutil.TempFile("", "filepipe")
	if err != nil {
		return nil, err
	}
	fp := &FilePipe{f: f}
	fp.cond.L = &fp.mu
	return fp, nil
}

// Release the resources associated with the filepipe. In particular,
// remove the backing file. After this call has been made no readers
// of this filepipe should be used.
func (fp *FilePipe) Release() error {
	fp.Close()
	filename := fp.f.Name()
	err := fp.f.Close()
	if err1 := os.Remove(filename); err == nil {
		err = err1
	}
	return err
}

// Create a new reader that starts reading the filepipe's contents from
// the very beginning.
func (fp *FilePipe) Reader() io.Reader {
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
