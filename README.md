parunner
========

Single-machine runner for [distributed](http://potyczki.mimuw.edu.pl/l/zadania_rozproszone/) [Potyczki Algorytmiczne](http://potyczki.mimuw.edu.pl/) problems ([a](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/mak/) [few](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/kol/) [examples](https://sio2.mimuw.edu.pl/pa/c/pa-2014-1/p/sek/)).

[![Build Status](https://drone.io/github.com/robryk/parunner/status.png)](https://drone.io/github.com/robryk/parunner/latest) [![GoDoc](https://godoc.org/github.com/robryk/parunner?status.png)](https://godoc.org/github.com/robryk/parunner)

Usage
-----

Link your binary with [zeus/zeus_local.c](https://github.com/robryk/parunner/blob/master/zeus/zeus_local.c) instead of the MPI-based zeus_local and pass the path to parunner. Parunner will explain its usage when called with no arguments.
