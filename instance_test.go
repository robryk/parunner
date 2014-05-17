package main

import (
	"os"
	"os/exec"
	"testing"
)

var trueBinary string

func init() {
	for _, s := range []string{"/bin/true"} {
		if _, err := os.Stat(s); err == nil {
			trueBinary = s
			break
		}
	}
}

func startInstance(cmd *exec.Cmd) (*Instance, error) {
	instances := make([]*Instance, 1)
	var err error
	instances[0], err = NewInstance(cmd, 0, instances)
	if err != nil {
		return nil, err
	}
	if err := instances[0].Start(); err != nil {
		return nil, err
	}
	return instances[0], nil
}

func TestInstanceSuccess(t *testing.T) {
	if trueBinary == "" {
		t.Skipf("No /bin/true equivalent found")
	}
	instance, err := startInstance(exec.Command(trueBinary))
	if err != nil {
		t.Fatalf("Error creating an instance for %s: %v", trueBinary, err)
	}
	if err := instance.Wait(); err != nil {
		t.Fatalf("Error running %s: %v", trueBinary, err)
	}
}
