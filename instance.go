package main

import (
	"io"
	"log"
	"os"
	"os/exec"
)

type Instance struct {
	id        int
	instances []*Instance
	cmd       *exec.Cmd

	messagesSent     int
	messageBytesSent int

	queues   []*MessageQueue
	selector chan *MessageQueue
}

// TODO: errors, communicate later, queues
func NewInstance(cmd *exec.Cmd, id int, instances []*Instance) (*Instance, error) {
	if cmd.Stdin != nil {
		stdin := cmd.Stdin
		cmd.Stdin = nil
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			io.Copy(stdinPipe, stdin) // XXX errors?
		}()
	}
	cmdr, cmdw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	respr, respw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.ExtraFiles = []*os.File{respr, cmdw} // XXX close these after Start()
	instance := &Instance{
		id:        id,
		instances: instances,
		cmd:       cmd,
		queues:    make([]*MessageQueue, len(instances)),
		selector:  make(chan *MessageQueue),
	}
	for i := range instance.queues {
		// XXX teardown
		instance.queues[i] = NewMessageQueue(instance.selector)
	}

	go instance.communicate(cmdr, respw) // XXX after init
	return instance, nil
}

func (i *Instance) Start() error {
	return i.cmd.Start()
}

func (i *Instance) Kill() error {
	return i.cmd.Process.Kill()
}
