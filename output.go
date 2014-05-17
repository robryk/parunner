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

func (cs *ContestStdout) Create(i int) io.Writer {
	pr, pw := io.Pipe()
	go func() {
		var buf [1]byte
		_, err := pr.Read(buf[:])
		if err != nil {
			// We want to quit on EOF and ignore errors
			return
		}
		cs.chooseInstance.Do(func() {
			cs.chosenInstance = i
		})
		if cs.chosenInstance == i {
			if _, err := cs.Output.Write(buf[:]); err != nil {
				return
			}
			io.Copy(cs.Output, pr)
		} else {
			pr.CloseWithError(fmt.Errorf("instancja %d zaczęła już wypisywać wyjście", cs.chosenInstance))
		}
	}()
	return pw
}

func TagStream(tag string, w io.Writer) io.Writer {
	pr, pw := io.Pipe()
	go func() {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			fmt.Fprintf(w, "%s%s\n", tag, sc.Text())
		}
	}()
	return pw
}
