package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func (i *Instance) getUtime() (time.Duration, error) {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(i.cmd.Process.Pid, &rusage); err != nil {
		return 0, err
	}
	return time.Duration(rusage.Utime.Nano()) * time.Nanosecond, nil
}

func setupPipes(cmd *exec.Cmd, r *os.File, w *os.File) {
	cmd.ExtraFiles = []*os.File{r, w}
}
