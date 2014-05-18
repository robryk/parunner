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

func (cs *ContestStdout) Write(i int, r io.Reader) error {
	var buf [1]byte
	_, err := r.Read(buf[:])
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return err
	}
	cs.chooseInstance.Do(func() {
		cs.chosenInstance = i
	})
	if cs.chosenInstance == i {
		if _, err := cs.Output.Write(buf[:]); err != nil {
			return err
		}
		_, err := io.Copy(cs.Output, r)
		return err
	} else {
		return fmt.Errorf("instancja %d zaczęła już wypisywać wyjście", cs.chosenInstance)
	}
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
