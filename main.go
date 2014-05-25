package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"text/tabwriter"
	"time"
)

const MaxInstances = 100

var nInstances = flag.Int("n", 1, fmt.Sprintf("Liczba instancji, z zakresu [1,%d]", MaxInstances))
var stdoutHandling = flag.String("stdout", "contest", "Obługa standardowego wyjścia: contest, all, tagged, files")
var stderrHandling = flag.String("stderr", "all", "Obsługa standardowe wyjścia diagnostycznego: all, tagged, files")
var filesPrefix = flag.String("prefix", "", "Prefiks nazwy plików wyjściowych generowanych przez -stdout=files i -stderr=files")
var warnRemaining = flag.Bool("warn_unreceived", true, "Ostrzegaj o wiadomościach, które pozostały nieodebrane po zakończeniu się instancji")
var stats = flag.Bool("print_stats", false, "Na koniec wypisz statystyki dotyczące poszczególnych instancji")

var binaryPath string

func writeFile(streamType string, i int, r io.Reader) error {
	basename := binaryPath // XXX -- remove extension
	if *filesPrefix != "" {
		basename = *filesPrefix
	}
	filename := fmt.Sprintf("%s.%s.%d", basename, streamType, i)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Uzycie: %s [opcje] program_do_uruchomienia\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `Sposoby obsługi wyjścia:
  contest: Wymuszaj, żeby tylko jedna instancja pisała na standardowe wyjście. Przekieruj jej wyjście na standardowe wyjście tego programu.
  all: Przekieruj wyjście wszystkich instancji na analogiczne wyjście tego programu.
  tagged: Przekieruj wyjście wszystkich instancji na analogiczne wyjście tego programy, dopisując numer instancji na początku każdej linijki.
  files: Zapisz wyjście każdej instancji w osobnym pliku.
`)
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Nie podałeś programu do uruchomienia\n")
		flag.Usage()
		os.Exit(1)
	}
	binaryPath = flag.Arg(0)

	if *nInstances < 1 || *nInstances > MaxInstances {
		fmt.Fprintf(os.Stderr, "Liczba instancji powinna być z zakresu [1,%d], a podałeś %d\n", MaxInstances, *nInstances)
		flag.Usage()
		os.Exit(1)
	}
	var writeStdout func(int, io.Reader) error
	contestStdout := &ContestStdout{Output: os.Stdout}
	switch *stdoutHandling {
	case "contest":
		// This is handled specially (without a pipe) below.
	case "all":
	case "tagged":
		writeStdout = func(i int, r io.Reader) error { return TagStream(fmt.Sprintf("STDOUT %d: ", i), os.Stdout, r) }
	case "files":
		writeStdout = func(i int, r io.Reader) error { return writeFile("stdout", i, r) }
	default:
		fmt.Fprintf(os.Stderr, "Niewłaściwa metoda obsługi standardowego wyjścia: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}
	var writeStderr func(int, io.Reader) error
	switch *stderrHandling {
	case "all":
	case "tagged":
		writeStderr = func(i int, r io.Reader) error { return TagStream(fmt.Sprintf("STDERR %d: ", i), os.Stderr, r) }
	case "files":
		writeStdout = func(i int, r io.Reader) error { return writeFile("stderr", i, r) }
	default:
		fmt.Fprintf(os.Stderr, "Niewłaściwa metoda obsługi standardowego wyjścia diagnostycznego: %s", *stdoutHandling)
		flag.Usage()
		os.Exit(1)
	}

	stdinPipe, err := NewFilePipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdinPipe.Release()
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
	progs := make([]*exec.Cmd, *nInstances)
	var wg sync.WaitGroup
	closeAfterWait := []io.Closer{}
	for i := range progs {
		cmd := exec.Command(flag.Arg(0))
		w, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			// We don't care about errors from the writer (we expect broken pipe if the process has exited
			// before reading all of its input), but we do care about errors when reading from the filepipe.
			if _, err := io.Copy(WrapWriter(w), stdinPipe.Reader()); err != nil {
				if _, ok := err.(WriterError); !ok {
					log.Fatal(err)
				}
			}
			w.Close()
		}()
		makeFromWrite := func(writeProc func(int, io.Reader) error, w io.Writer) io.Writer {
			if writeProc == nil {
				return w
			}
			pr, pw := io.Pipe()
			closeAfterWait = append(closeAfterWait, pw)
			i := i
			wg.Add(1)
			go func() {
				err := writeProc(i, pr)
				if err != nil {
					// All the errors we can get are not caused by instances' invalid behaviour, but
					// by system issues (can't create a file, broken pipe on real stdout/err, etc.)
					log.Fatal(err)
				}
				wg.Done()
			}()
			return pw
		}
		if *stdoutHandling == "contest" {
			cmd.Stdout = contestStdout.NewWriter(i)
		} else {
			cmd.Stdout = makeFromWrite(writeStdout, os.Stdout)
		}
		cmd.Stderr = makeFromWrite(writeStderr, os.Stderr)
		progs[i] = cmd
	}
	instances, err := RunInstances(progs)
	for _, f := range closeAfterWait {
		f.Close()
	}
	wg.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i, instance := range instances {
		buf := instance.ShutdownQueues()
		if *warnRemaining && len(buf) > 0 {
			fmt.Fprintf(os.Stderr, "Uwaga: Instancja %d nie odebrała %d wiadomości dla niej przeznaczonych przed swoim zakończeniem.\n", i, len(buf))
		}
	}
	var maxTime time.Duration
	var lastInstance int
	for i, instance := range instances {
		if instanceTime := instance.TimeRunning + instance.TimeBlocked; instanceTime >= maxTime {
			maxTime = instanceTime
			lastInstance = i
		}
	}
	fmt.Fprintf(os.Stderr, "Czas trwania: %v (najdłużej działająca instancja: %d)\n", maxTime, lastInstance)
	if *stats {
		w := tabwriter.NewWriter(os.Stderr, 2, 1, 1, ' ', 0)
		io.WriteString(w, "Instancja\tCzas całkowity\tCzas CPU\tCzas oczekiwania\tWysłane wiadomości\tWysłane bajty\n")
		for i, instance := range instances {
			fmt.Fprintf(w, "%d\t%v\t%v\t%v\t%d\t%d\n", i, instance.TimeRunning + instance.TimeBlocked, instance.TimeRunning, instance.TimeBlocked, instance.MessagesSent, instance.MessageBytesSent)
		}
		w.Flush()
	}
}
