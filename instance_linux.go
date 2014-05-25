package main

import (
	"os"
	"os/exec"
)

func setupPipes(cmd *exec.Cmd, r *os.File, w *os.File) {
	cmd.ExtraFiles = []*os.File{r, w}
}

// TODO: windows version
