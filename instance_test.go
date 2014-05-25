package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
		t.Skip("no /bin/true equivalent found")
	}
	instance := &Instance{ID: 0, TotalInstances: 1, Cmd: exec.Command(trueBinary)}
	if err := instance.Start(); err != nil {
		t.Fatalf("error starting an instance of %s: %v", trueBinary, err)
	}
	if err := checkedWait(t, instance); err != nil {
		t.Fatalf("error running %s: %v", trueBinary, err)
	}
}

func TestInstanceFailure(t *testing.T) {
	if falseBinary == "" {
		t.Skip("no /bin/false equivalent found")
	}
	instance := &Instance{ID: 0, TotalInstances: 1, Cmd: exec.Command(falseBinary)}
	if err := instance.Start(); err != nil {
		t.Fatalf("error starting an instance of %s: %v", falseBinary, err)
	}
	if err := checkedWait(t, instance); err == nil {
		t.Fatalf("no error when running %s", falseBinary)
	}
}

func TestInstanceKill(t *testing.T) {
	if catBinary == "" {
		t.Skip("no /bin/cat equivalent found")
	}
	cmd := exec.Command(catBinary)
	if _, err := cmd.StdinPipe(); err != nil {
		t.Fatalf("error in Cmd.StdinPipe: %v", err)
	}
	cmd.Stdout = ioutil.Discard
	instance := &Instance{ID: 0, TotalInstances: 1, Cmd: cmd}
	if err := instance.Start(); err != nil {
		t.Fatalf("error starting an instance of %s: %v", falseBinary, err)
	}
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
		t.Errorf("a killed instance has finished with error %v, instead of %v", err, ErrKilled)
	}
}

func TestInstanceComm(t *testing.T) {
	testerBinary := filepath.Join("zeus", "tester")
	if _, err := os.Stat(testerBinary); err != nil {
		t.Skipf("%s not found", testerBinary)
	}
	type testcase struct {
		name             string
		input            string
		expectedOutput   string
		expectedRequests []*request // expectedRequests[].time is a lower bound on the actual time
		responses        []*response
	}
	singleCase := func(tc testcase) {
		quit := make(chan bool)

		cmd := exec.Command(testerBinary)
		cmd.Stdin = strings.NewReader(tc.input)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		instance := &Instance{
			ID:             5,
			TotalInstances: 20,
			Cmd:            cmd,
			RequestChan:    make(chan *request, 1),
			ResponseChan:   make(chan *response, 1),
		}
		if err := instance.Start(); err != nil {
			t.Errorf("test %s: error starting an instance of %s: %v", tc.name, testerBinary, err)
			return
		}

		go func() {
			var prevTime time.Duration
			var i int
			for req := range instance.RequestChan {
				if req.time < prevTime {
					t.Errorf("test %s: request %+v is earlier than %v, the time of the previous request", tc.name, req, prevTime)
				}
				if i < len(tc.expectedRequests) {
					if req.time < tc.expectedRequests[i].time {
						t.Errorf("test %s: request %+v has time %v, expected at least %v", tc.name, req, req.time, tc.expectedRequests[i].time)
					}
					realTime := req.time
					req.time = tc.expectedRequests[i].time
					if got, want := req, tc.expectedRequests[i]; !reflect.DeepEqual(got, want) {
						got.time = realTime
						t.Errorf("test %s: got request %+v, expected %+v", tc.name, got, want)
					}
				} else {
					t.Errorf("test %s: got request number %d, expected %d total", tc.name, i, len(tc.expectedRequests))
				}
				i++
			}
			if i < len(tc.expectedRequests) {
				t.Errorf("test %s: got only %d requests, expected %d", tc.name, i, len(tc.expectedRequests))
			}
			<-quit
		}()
		go func() {
			for i, resp := range tc.responses {
				select {
				case instance.ResponseChan <- resp:
				case <-quit:
					t.Errorf("test %s: instance was done before receiving response number %d", tc.name, i)
					return
				}
			}
			<-quit
		}()
		defer func() {
			quit <- true
			quit <- true
		}()

		if err := checkedWait(t, instance); err != nil {
			t.Fatalf("test %s: error running an instance of %s: %v", tc.name, testerBinary, err)
			return
		}
		close(instance.RequestChan)
		if got, want := stdout.String(), tc.expectedOutput; got != want {
			t.Errorf("test %s: wrong output; got=%q, want=%q", tc.name, got, want)
		}
	}
	testcases := []testcase{
		{"header", "", "5 20\n", []*request{}, []*response{}},
		{"send", "Scfoobar\n", "5 20\n", []*request{&request{requestType: requestSend, destination: 2, message: []byte("foobar")}}, []*response{}},
		{"recv", "Rd\n", "5 20\n3 6 foobaz\n", []*request{&request{requestType: requestRecv, source: 3}}, []*response{&response{&Message{Source: 3, Target: 5, Message: []byte("foobaz")}}}},
		{"recvany", "R*\n", "5 20\n3 6 foobaz\n", []*request{&request{requestType: requestRecvAny}}, []*response{&response{&Message{Source: 3, Target: 5, Message: []byte("foobaz")}}}},
		{"blockingTime", "R*\nScblah\n", "5 20\n3 6 foobaz\n", []*request{
			&request{requestType: requestRecvAny},
			&request{requestType: requestSend, time: time.Duration(1234), destination: 2, message: []byte("blah")},
		}, []*response{&response{&Message{Source: 3, Target: 5, SendTime: time.Duration(1234), Message: []byte("foobaz")}}}},
	}

	for _, tc := range testcases {
		singleCase(tc)
	}
}

// Stop receiving in the middle of a message
func TestInstanceBrokenPipe(t *testing.T) {
	hangerBinary := filepath.Join("zeus", "hanger")
	if _, err := os.Stat(hangerBinary); err != nil {
		t.Skipf("%s not found", hangerBinary)
	}
	cmd := exec.Command(hangerBinary)
	instance := &Instance{
		ID:             0,
		TotalInstances: 2,
		Cmd:            cmd,
		RequestChan:    make(chan *request, 1),
		ResponseChan:   make(chan *response, 1),
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("error starting an instance of %s: %v", hangerBinary, err)
	}
	go func() {
		for _ = range instance.RequestChan {
		}
	}()
	defer close(instance.RequestChan)
	instance.ResponseChan <- &response{&Message{
		Source:   1,
		Target:   0,
		SendTime: time.Duration(0),
		Message:  []byte("abcdefghijlkmnopqrstuvwxyz"), // this message will take >20 bytes on the wire
	}}
	if err := checkedWait(t, instance); err != nil {
		t.Fatalf("error running an instance of %s: %v", hangerBinary, err)
	}
}
