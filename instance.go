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

	errChan chan error
	errOnce sync.Once
	err     error
}

// TODO: errors, communicate later, queues
func NewInstance(cmd *exec.Cmd, id int, totalInstances int, outgoingMessages chan<- Message) (*Instance, error) {
	instance := &Instance{
		id:               id,
		totalInstances:   totalInstances,
		outgoingMessages: outgoingMessages,
		cmd:              cmd,
		queues:           make([]*MessageQueue, totalInstances),
		selector:         make(chan *MessageQueue),
		errChan:          make(chan error, 4),
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
	for i := range instance.queues {
		// XXX teardown
		instance.queues[i] = NewMessageQueue(instance.selector)
	}

	instance.communicateGoroutine = func() {
		if err := instance.communicate(cmdr, respw); err != nil {
			select {
			case instance.errChan <- err:
			default:
			}
			instance.cmd.Process.Kill()
		}
		cmdr.Close()
		respw.Close()
	}
	return instance, nil
}

func (i *Instance) Start() error {
	if err := i.cmd.Start(); err != nil {
		return err
	}
	go i.communicateGoroutine()
	go func() {
		select {
		case i.errChan <- i.cmd.Wait():
		default:
		}
		// We are doing it this late in order to delay error reports from communicate that are
		// a result of the pipes closing (broken pipe on write pipe, EOF on read pipe). We
		// do want to ignore some of those errors (e.g. broken pipe at the very beginning, which
		// indicates that the program didn't use the communication library at all), so currently
		// we ignore all of them.
		// TODO: Do we want to ignore then also when the program has terminated with no errors?
		//       Example: program has exited in the middle of sending a message.
		for _, f := range i.cmd.ExtraFiles {
			// TODO: should we ignore the error here?
			f.Close()
		}
	}()
	return nil
}

func (i *Instance) Wait() error {
	i.errOnce.Do(func() {
		i.err = <-i.errChan
	})
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
	select {
	case i.errChan <- ErrKilled:
	default:
	}
	return i.cmd.Process.Kill()
}
