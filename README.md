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
2. Obtain a binary of parunner. If you have a Go toolchain installed, you can compile it by doing `go install github.com/robryk/parunner`. There is also a compiled binary for [linux-amd64](https://drone.io/github.com/robryk/parunner/files/parunner) available.
3. Run `parunner -n=5 -trace_comm -stdout=all zeus/example`

For more information on parunner's usage invoke it with no arguments.
