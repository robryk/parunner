parunner
========

Single-machine runner for [distributed](http://potyczki.mimuw.edu.pl/l/zadania_rozproszone/) [Potyczki Algorytmiczne](http://potyczki.mimuw.edu.pl/) problems ([a](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/mak/) [few](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/kol/) [examples](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/sek/)).

[![Build Status](https://drone.io/github.com/robryk/parunner/status.png)](https://drone.io/github.com/robryk/parunner/latest) [![GoDoc](https://godoc.org/github.com/robryk/parunner?status.png)](https://godoc.org/github.com/robryk/parunner)

Usage
-----

In order to run a program that uses [raw zeus interface](https://github.com/robryk/parunner/blob/master/zeus/zeus.h), you need to link it with [zeus/zeus_local.c](https://github.com/robryk/parunner/blob/master/zeus/zeus_local.c) instead of any other implementation of zeus_local. You can then run the program as follows:

    $ parunner -n=number_of_instances path/to/program

There is an [example](https://github.com/robryk/parunner/blob/master/zeus/example.c) provided. In order to run it, you should:

1. Compile it: `make -C zeus example`
2. Obtain a binary of parunner. If you have a Go toolchain installed, you can compile it by doing `go get github.com/robryk/parunner`. The binary will then be built and written to `$GOPATH/bin/parunner`. There is also a compiled binary for [linux-amd64](https://drone.io/github.com/robryk/parunner/files/parunner) available.
3. Run `parunner -n=3 -trace_comm -stdout=tagged zeus/example`. The output should look like this:
```
$ parunner -n=3 -trace_comm -stdout=tagged zeus/example
STDOUT 0: #nodes is 3, and I have the number 0.
STDOUT 0: sends a message to 1.
STDOUT 1: #nodes is 3, and I have the number 1.
STDOUT 1: Sends a message to the 2.
STDOUT 1: I receive a message from 0.
STDOUT 2: #nodes is 3, and I have a number 2.
STDOUT 2: I receive a message from the 1.
COMM: 1 instance instance of 0 sends me a message (13 bytes) [0]
COMM instance of 2: 1 instance sends me a message (13 bytes) [0]
COMM: 1 instance: I'm waiting for a message from an instance of 0 [0]
COMM: 1 instance: I received a message from an instance of 0 (13 bytes)
STDOUT 1: I picked: Hello from 0!
COMM: 2 instance: I'm waiting for a message from instance 1 [0]
COMM: 2 instance: I received a message from one instance (13 bytes)
STDOUT 2: I picked: Hello from 1!
Duration: 0 (longest running instance: 2)
```

For more information on parunner's usage invoke it with no arguments.
