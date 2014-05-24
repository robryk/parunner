package main

import (
	"errors"
	"os"
	"os/exec"
	"sync"
)

type Instance struct {
	id             int
	totalInstances int
	cmd            *exec.Cmd

	requestChan  chan *request
	responseChan chan *response

	messagesSent     int
	messageBytesSent int

	errOnce  sync.Once
	err      error
	waitDone chan bool
	commDone chan bool
}

func StartInstance(cmd *exec.Cmd, id int, totalInstances int) (*Instance, error) {
	instance := &Instance{
		id:             id,
		totalInstances: totalInstances,
		cmd:            cmd,
		requestChan:    make(chan *request, 1),
		responseChan:   make(chan *response, 1),
		waitDone:       make(chan bool),
		commDone:       make(chan bool),
	}
	cmdr, cmdw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	respr, respw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	setupPipes(cmd, respr, cmdw)

	if err := instance.cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		if err := instance.communicate(cmdr, respw, instance.requestChan, instance.responseChan); err != nil {
			instance.errOnce.Do(func() {
				instance.err = err
			})
			instance.cmd.Process.Kill()
		}
		cmdr.Close()
		respw.Close()
		close(instance.commDone)
	}()
	go func() {
		err := instance.cmd.Wait()
		instance.errOnce.Do(func() {
			instance.err = err
		})
		// We are doing it this late in order to delay error reports from communicate that are
		// a result of the pipes closing (broken pipe on write pipe, EOF on read pipe). We
		// do want to ignore some of those errors (e.g. broken pipe at the very beginning, which
		// indicates that the program didn't use the communication library at all), so currently
		// we ignore all of them.
		// TODO: Do we want to ignore then also when the program has terminated with no errors?
		//       Example: program has exited in the middle of sending a message.
		respr.Close()
		cmdw.Close()
		close(instance.waitDone)
	}()
	return instance, nil
}

func (i *Instance) Wait() error {
	<-i.waitDone
	<-i.commDone
	return i.err
}

func (i *Instance) ShutdownQueues() []Message {
	buf := []Message(nil)
	// TODO
	return buf
}

var ErrKilled = errors.New("killed by an explicit request")

func (i *Instance) Kill() error {
	i.errOnce.Do(func() {
		i.err = ErrKilled
	})
	return i.cmd.Process.Kill()
}
