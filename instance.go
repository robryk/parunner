package main

import (
	"errors"
	"os"
	"os/exec"
	"sync"
)

type Instance struct {
	id               int
	totalInstances   int
	outgoingMessages chan<- Message
	cmd              *exec.Cmd

	communicateGoroutine func()

	messagesSent     int
	messageBytesSent int

	queues   []*MessageQueue
	selector chan *MessageQueue

	errOnce sync.Once
	err     error
	wg      sync.WaitGroup
}

func StartInstance(cmd *exec.Cmd, id int, totalInstances int, outgoingMessages chan<- Message) (*Instance, error) {
	instance := &Instance{
		id:               id,
		totalInstances:   totalInstances,
		outgoingMessages: outgoingMessages,
		cmd:              cmd,
		queues:           make([]*MessageQueue, totalInstances),
		selector:         make(chan *MessageQueue),
	}
	cmdr, cmdw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	respr, respw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.ExtraFiles = []*os.File{respr, cmdw}

	if err := instance.cmd.Start(); err != nil {
		return nil, err
	}

	for i := range instance.queues {
		instance.queues[i] = NewMessageQueue(instance.selector)
	}
	instance.wg.Add(1)
	go func() {
		if err := instance.communicate(cmdr, respw); err != nil {
			instance.errOnce.Do(func() {
				instance.err = err
			})
			instance.cmd.Process.Kill()
		}
		cmdr.Close()
		respw.Close()
		instance.wg.Done()
	}()
	instance.wg.Add(1)
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
		instance.wg.Done()
	}()
	return instance, nil
}

func (i *Instance) Wait() error {
	i.wg.Wait()
	return i.err
}

func (i *Instance) ShutdownQueues() []Message {
	buf := []Message(nil)
	for _, q := range i.queues {
		for _, m := range q.Shutdown() {
			buf = append(buf, m.(Message))
		}
	}
	return buf
}

var ErrKilled = errors.New("killed by an explicit request")

func (i *Instance) Kill() error {
	i.errOnce.Do(func() {
		i.err = ErrKilled
	})
	return i.cmd.Process.Kill()
}
