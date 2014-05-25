package main

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

type ContestStdout struct {
	Output         io.Writer
	chosenInstance int
	chooseInstance sync.Once
}

type contestStdoutWriter struct {
	cs *ContestStdout
	id int
}

func (w *contestStdoutWriter) Write(buf []byte) (int, error) {
	w.cs.chooseInstance.Do(func() {
		w.cs.chosenInstance = w.id
	})
	if w.cs.chosenInstance == w.id {
		return w.cs.Output.Write(buf)
	} else {
		return 0, fmt.Errorf("instancja %d zaczęła już wypisywać wyjście", w.cs.chosenInstance)
	}
}

func (cs *ContestStdout) NewWriter(id int) io.Writer {
	return &contestStdoutWriter{cs: cs, id: id}
}

func TagStream(tag string, w io.Writer, r io.Reader) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if _, err := fmt.Fprintf(w, "%s%s\n", tag, sc.Text()); err != nil {
			return err
		}
	}
	return sc.Err()
}
