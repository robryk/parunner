package main

import (
	"fmt"
	"sync"
)

type Instances []*Instance

// An error with the ID of the instance concerned attached.
type InstanceError struct {
	ID  int
	Err error
}

func (ie InstanceError) Error() string {
	return fmt.Sprintf("Błąd instancji %d: %v", ie.ID, ie.Err)
}

// Run runs all the instances and waits for them all to terminate. If an instance
// fails, Run kills the rest of the instances. Returns an InstanceError that wraps
// the first error encountered.
func (is Instances) Run() error {
	var wg sync.WaitGroup
	results := make(chan error, 1)
	for i, instance := range is {
		wg.Add(1)
		if err := instance.Start(); err != nil {
			return InstanceError{i, err}
		}
		defer func(instance *Instance) {
			instance.Kill()
			instance.Wait()
		}(instance)
		go func(i int, instance *Instance) {
			err := instance.Wait()
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
	return <-results
}
