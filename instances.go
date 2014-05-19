package main

import (
	"fmt"
	"os/exec"
	"sync"
)

// InstanceError is an error with an associated instance ID
type InstanceError struct {
	ID  int
	Err error
}

func (ie InstanceError) Error() string {
	return fmt.Sprintf("Błąd instancji %d: %v", ie.ID, ie.Err)
}

// RunInstances starts each command from cmds in an Instance and
// waits either for all of them to finish successfully or for
// the first error. In the latter case, all the rest of
// the instances are killed. All the instances are then returned
// in the slice. RunInstances additionally guarantees the following:
// * The instance slice is valid even if the error is non-nil
// * All the commands have been started before RunInstances returns
// * All the instanced have been waited on before RunInstances returns
// * The error returned is an instance of InstanceError and contains
//   the instance ID that caused the error.
func RunInstances(cmds []*exec.Cmd) ([]*Instance, error) {
	var wg sync.WaitGroup
	defer wg.Wait()

	messagesCh := make(chan Message, 1)
	results := make(chan error, 1)
	is := make([]*Instance, len(cmds))
	for i, cmd := range cmds {
		var err error
		is[i], err = StartInstance(cmd, i, len(cmds), messagesCh)
		if err != nil {
			is[i] = nil
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
		wg.Wait()
		select {
		case results <- nil:
		default:
		}
	}()
	for {
		select {
		case m := <-messagesCh:
			is[m.Target].PutMessage(m)
		case err := <-results:
			return is, err
		}
	}
}
