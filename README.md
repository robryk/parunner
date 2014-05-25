parunner
========

Single-machine runner for [distributed](http://potyczki.mimuw.edu.pl/l/zadania_rozproszone/) [Potyczki Algorytmiczne](http://potyczki.mimuw.edu.pl/) problems.

[![Build Status](https://drone.io/github.com/robryk/parunner/status.png)](https://drone.io/github.com/robryk/parunner/latest) [![GoDoc](https://godoc.org/github.com/robryk/parunner?status.png)](https://godoc.org/github.com/robryk/parunner)

Usage
-----

Link your binary with [zeus/zeus_local.c](https://github.com/robryk/parunner/blob/master/zeus/zeus_local.c) instead of the MPI-based zeus_local and pass the path to parunner. Parunner will explain its usage when called with no arguments.
