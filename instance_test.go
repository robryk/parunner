package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	instance, err := StartInstance(cmd, 0, 1, nil)
	if err != nil {
		t.Fatalf("Error starting an instance for %s: %v", cmd.Path, err)
	}
	return instance
}

func checkedWait(t *testing.T, instance *Instance) error {
	ch := make(chan error, 1)
	go func() {
		ch <- instance.Wait()
	}()
	err := instance.Wait()
	if err1 := <-ch; err1 != err {
		t.Errorf("Instance.Wait() gave contradictory return values: %v != %v", err, err1)
	}
	return err
}

func TestInstanceSuccess(t *testing.T) {
	if trueBinary == "" {
		t.Skip("No /bin/true equivalent found")
	}
	instance := startInstance(t, exec.Command(trueBinary))
	if err := checkedWait(t, instance); err != nil {
		t.Fatalf("Error running %s: %v", trueBinary, err)
	}
}

func TestInstanceFailure(t *testing.T) {
	if falseBinary == "" {
		t.Skip("No /bin/false equivalent found")
	}
	instance := startInstance(t, exec.Command(falseBinary))
	if err := checkedWait(t, instance); err == nil {
		t.Fatalf("No error when running %s", falseBinary)
	}
}

func TestInstanceKill(t *testing.T) {
	if catBinary == "" {
		t.Skip("No /bin/cat equivalent found")
	}
	cmd := exec.Command(catBinary)
	if _, err := cmd.StdinPipe(); err != nil {
		t.Fatalf("error in Cmd.StdinPipe: %v", err)
	}
	cmd.Stdout = ioutil.Discard
	instance := startInstance(t, cmd)
	waitChan := make(chan error)
	go func() {
		waitChan <- checkedWait(t, instance)
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

func TestInstanceComm(t *testing.T) {
	testerBinary := filepath.Join("zeus", "tester")
	const inputText = ``
	const outputText = `5 20
`
	if _, err := os.Stat(testerBinary); err != nil {
		t.Skipf("%s not found", testerBinary)
	}
	cmd := exec.Command(testerBinary)
	cmd.Stdin = strings.NewReader(inputText)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	messageCh := make(chan Message, 10)
	instance, err := StartInstance(cmd, 5, 20, messageCh)
	if err != nil {
		t.Fatalf("error starting an instance with %s: %v", testerBinary, err)
	}
	if err := checkedWait(t, instance); err != nil {
		t.Fatalf("error running an instance with %s: %v", testerBinary, err)
	}
	if got, want := stdout.String(), outputText; got != want {
		t.Errorf("wrong output; got=\"%s\", want=\"%s\"", got, want)
	}
	// TODO: test message send and receive
}

// Stop receiving in the middle of a message
func TestInstanceBrokenPipe(t *testing.T) {
	hangerBinary := filepath.Join("zeus", "hanger")
	if _, err := os.Stat(hangerBinary); err != nil {
		t.Skipf("%s not found", hangerBinary)
	}
	cmd := exec.Command(hangerBinary)
	instance, err := StartInstance(cmd, 0, 2, nil)
	if err != nil {
		t.Fatalf("error starting an instance with %s: %v", hangerBinary, err)
	}
	instance.PutMessage(Message{Source: 1, Target: 0, Message: []byte("abcdefghijlkmnopqrstuvwxyz")}) // this message will take >20 bytes on the wire
	if err := checkedWait(t, instance); err != nil {
		t.Fatalf("error running an instance with %s: %v", hangerBinary, err)
	}
}
