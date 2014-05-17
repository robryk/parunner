package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"
)

var catBinary string
var trueBinary string
var falseBinary string

func init() {
	for _, s := range []string{"/bin/true"} {
		if _, err := os.Stat(s); err == nil {
			trueBinary = s
			break
		}
	}
	for _, s := range []string{"/bin/false"} {
		if _, err := os.Stat(s); err == nil {
			falseBinary = s
			break
		}
	}
	for _, s := range []string{"/bin/cat"} {
		if _, err := os.Stat(s); err == nil {
			catBinary = s
			break
		}
	}
}

func startInstance(t *testing.T, cmd *exec.Cmd) *Instance {
	instances := make([]*Instance, 1)
	var err error
	instances[0], err = NewInstance(cmd, 0, instances)
	if err != nil {
		t.Fatalf("Error creating an instance for %s: %v", cmd.Path, err)
	}
	if err := instances[0].Start(); err != nil {
		t.Fatalf("Error starting an instance for %s: %v", cmd.Path, err)
	}
	return instances[0]
}

func TestInstanceSuccess(t *testing.T) {
	if trueBinary == "" {
		t.Skip("No /bin/true equivalent found")
	}
	instance := startInstance(t, exec.Command(trueBinary))
	if err := instance.Wait(); err != nil {
		t.Fatalf("Error running %s: %v", trueBinary, err)
	}
}

func TestInstanceFailure(t *testing.T) {
	if falseBinary == "" {
		t.Skip("No /bin/false equivalent found")
	}
	instance := startInstance(t, exec.Command(falseBinary))
	if err := instance.Wait(); err == nil {
		t.Fatalf("No error when running %s", falseBinary)
	}
}

func TestInstanceKill(t *testing.T) {
	if catBinary == "" {
		t.Skip("No /bin/cat equivalent found")
	}
	cmd := exec.Command(catBinary)
	r, _ := io.Pipe()
	defer r.Close()
	cmd.Stdin = r
	cmd.Stdout = ioutil.Discard
	instance := startInstance(t, cmd)
	waitChan := make(chan error)
	go func() {
		waitChan <- instance.Wait()
	}()
	// The instance shouldn't finish of its own accord
	select {
	case err := <-waitChan:
		t.Fatalf("/bin/cat has finished prematurely, err=%v", err)
	case <-time.After(100 * time.Millisecond):
	}
	instance.Kill()
	if err := <-waitChan; err != ErrKilled {
		t.Errorf("A killed instance has finished with error %v, instead of %v", err, ErrKilled)
	}
}
