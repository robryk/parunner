package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

const MaxInstances = 100

var nInstances = flag.Int("n", 1, fmt.Sprintf("Liczba instancji, z zakresu [1,%d]", MaxInstances))
var stdoutHandling = flag.String("stdout", "contest", "Obługa standardowego wyjścia: contest, all, tagged, files")
var stderrHandling = flag.String("stderr", "all", "Obsługa standardowe wyjścia diagnostycznego: all, tagged, files")
var filesPrefix = flag.String("prefix", "", "Prefiks nazwy plików wyjściowych generowanych przez -stdout=files i -stderr=files")
var binaryPath string

func outputFile(streamType string, i int) *os.File {
	basename := binaryPath // XXX -- remove extension
	if *filesPrefix != "" {
		basename = *filesPrefix
	}
	filename := fmt.Sprintf("%s.%s.%d", basename, streamType, i)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err) // XXX
	}
	// XXX close them sometime
	return file
}
func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Podaj nazwe programu XXX")
		os.Exit(1)
	}
	binaryPath = flag.Arg(0)
	if *nInstances < 1 || *nInstances > MaxInstances {
		fmt.Fprintf(os.Stderr, "Liczba instancji powinna być z zakresu [1,100], a podałeś %d\n", *nInstances)
		flag.Usage()
		os.Exit(1)
	}
	var makeStdout func(int) io.Writer
	switch *stdoutHandling {
	case "contest":
		cs := ContestStdout{Output: os.Stdout}
		makeStdout = cs.Create
	case "all":
		makeStdout = func(int) io.Writer { return os.Stdout }
	case "tagged":
		makeStdout = func(i int) io.Writer { return TagStream(fmt.Sprintf("STDOUT %d: ", i), os.Stdout) }
	case "files":
		makeStdout = func(i int) io.Writer { return outputFile("stdout", i) }
	default:
		fmt.Fprintf(os.Stderr, "Niewłaściwa metoda obsługi standardowego wyjścia: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}

	var makeStderr func(int) io.Writer
	switch *stderrHandling {
	case "all":
		makeStderr = func(int) io.Writer { return os.Stderr }
	case "tagged":
		makeStderr = func(i int) io.Writer { return TagStream(fmt.Sprintf("STDERR %d: ", i), os.Stderr) }
	case "files":
		makeStderr = func(i int) io.Writer { return outputFile("stderr", i) }
	default:
		fmt.Fprintf(os.Stderr, "Niewłaściwa metoda obsługi standardowego wyjścia diagnostycznego: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}

	instances := make([]*Instance, *nInstances)
	stdinPipe, err := NewFilePipe()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		_, err := io.Copy(stdinPipe, os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		err = stdinPipe.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	// XXX check the binary first?
	for i := range instances {
		cmd := exec.Command(flag.Arg(0))
		cmd.Stdin = stdinPipe.Reader()
		cmd.Stdout = makeStdout(i)
		cmd.Stderr = makeStderr(i)
		instances[i], err = NewInstance(cmd, i, instances)
		if err != nil {
			log.Fatal(err)
		}
	}
	type idAndError struct {
		id  int
		err error
	}
	var wg sync.WaitGroup
	results := make(chan idAndError, 1)
	for i, instance := range instances {
		wg.Add(1)
		go func(i int, instance *Instance) {
			err := instance.Run()
			if err != nil {
				select {
				case results <- idAndError{i, err}:
				default:
				}
			}
			wg.Done()
		}(i, instance)
	}
	go func() {
		wg.Wait()
		select {
		case results <- idAndError{}:
		default:
		}
	}()
	firstError := <-results
	if firstError.err != nil {
		for _, instance := range instances {
			instance.Kill()
		}
		wg.Wait()
		fmt.Fprintf(os.Stderr, "Instancja %d zakończyła się z błędem: %v\n", firstError.id, firstError.err)
		os.Exit(1)
	}
}
