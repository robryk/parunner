package main

import (
	"fmt"
	"os/exec"
	"sync"
)

// An error with the ID of the instance concerned attached.
type InstanceError struct {
	ID  int
	Err error
}

func (ie InstanceError) Error() string {
	return fmt.Sprintf("Błąd instancji %d: %v", ie.ID, ie.Err)
}

func RunInstances(cmds []*exec.Cmd) ([]*Instance, error) {
	messagesCh := make(chan Message, 1)
	var wg sync.WaitGroup

	defer func() {
		wg.Wait()
		// This channel has to be closed _after_ all the instances have terminated,
		// because it's written to by the instances when they send a message.
		// Thus, we close it only after observing that wg is empty.
		close(messagesCh)
	}()

	results := make(chan error, 1)
	is := make([]*Instance, len(cmds))
	for i, cmd := range cmds {
		var err error
		is[i], err = StartInstance(cmd, i, len(cmds), messagesCh)
		if err != nil {
			select {
			case results <- InstanceError{i, err}:
			default:
			}
			continue
		}
		defer is[i].Kill()
		wg.Add(1)
		go func(i int, instance *Instance) {
			err := instance.Wait()
			if err != nil {
				select {
				case results <- InstanceError{i, err}:
				default:
				}
			}
			wg.Done()
		}(i, is[i])
	}
	go func() {
		for m := range messagesCh {
			is[m.Target].PutMessage(m)
		}
	}()
	go func() {
		wg.Wait()
		select {
		case results <- nil:
		default:
		}
	}()
	return is, <-results
}
