package main

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
)

type Instance struct {
	id        int
	instances []*Instance
	cmd       *exec.Cmd

	stdinCopier          func()
	communicateGoroutine func()

	messagesSent     int
	messageBytesSent int

	queues   []*MessageQueue
	selector chan *MessageQueue

	errChan chan error
}

// TODO: errors, communicate later, queues
func NewInstance(cmd *exec.Cmd, id int, instances []*Instance) (*Instance, error) {
	instance := &Instance{
		id:        id,
		instances: instances,
		cmd:       cmd,
		queues:    make([]*MessageQueue, len(instances)),
		selector:  make(chan *MessageQueue),
		errChan:   make(chan error, 4),
	}
	if cmd.Stdin != nil {
		stdin := cmd.Stdin
		cmd.Stdin = nil
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}
		instance.stdinCopier = func() {
			defer stdinPipe.Close()
			_, err := io.Copy(stdinPipe, stdin)
			if err != nil {
				select {
				case instance.errChan <- err:
				default:
				}
				instance.cmd.Process.Kill()
			}
		}
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

// Start the instance, wait until it exits and return the first error encountered while executing it.
// Any errors encountered will cause the instance to be terminated.
func (i *Instance) Run() error {
	if err := i.cmd.Start(); err != nil {
		return err
	}
	defer i.cmd.Process.Kill()
	for _, f := range i.cmd.ExtraFiles {
		if err := f.Close(); err != nil {
			return err
		}
	}

	if i.stdinCopier != nil {
		go i.stdinCopier()
	}
	go i.communicateGoroutine()
	go func() {
		select {
		case i.errChan <- i.cmd.Wait():
		default:
		}
	}()
	return <-i.errChan
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

var ErrKilled = errors.New("Killed by explicit request")

func (i *Instance) Kill() error {
	select {
	case i.errChan <- ErrKilled:
	default:
	}
	return i.cmd.Process.Kill()
}
