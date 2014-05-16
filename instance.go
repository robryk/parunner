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
}

// TODO: errors, communicate later, queues
func NewInstance(cmd *exec.Cmd, id int, instances []*Instance) (*Instance, error) {
	i := &Instance{
		id:        id,
		instances: instances,
		cmd:       cmd,
	}
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
	i.cmd.ExtraFiles = []*os.File{respr, cmdw}
	go i.communicate(cmdr, respw) // XXX after init
	return i, nil
}

func (i *Instance) Start() error {
	return i.cmd.Start()
}

func (i *Instance) Kill() error {
	return i.cmd.Process.Kill()
}
