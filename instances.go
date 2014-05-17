package main

import (
	"fmt"
	"sync"
)

type Instances []*Instance

// An error with an attached instance ID, where it has occurred.
type InstanceError struct {
	Id  int
	Err error
}

func (ie InstanceError) Error() string {
	return fmt.Sprintf("Błąd instancji %d: %v", ie.Id, ie.Err)
}

// Run runs all the instances and wait for them all to terminate. If an instance fails,
// run kills the rest of the instances. Returns an InstanceError that wraps the first
// error encountered.
func (is Instances) Run() error {
	var wg sync.WaitGroup
	results := make(chan error, 1)
	for i, instance := range is {
		wg.Add(1)
		go func(i int, instance *Instance) {
			err := instance.Run()
			if err != nil {
				select {
				case results <- InstanceError{i, err}:
				default:
				}
			}
			wg.Done()
		}(i, instance)
	}
	go func() {
		wg.Wait()
		select {
		case results <- nil:
		default:
		}
	}()
	firstError := <-results
	if firstError != nil {
		for _, instance := range is {
			instance.Kill()
		}
	}
	wg.Wait()
	return firstError
}
