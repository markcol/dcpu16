DCPU-16 Vitual Machine tools written in Go
==========================================

[![Build Status](https://travis-ci.org/markcol/dcpu16.png)](https://travis-ci.org/markcol/dcpu16)

This is an implementation of the
[DCPU-16 specification](http://0x10c.com/doc/dcpu-16.txt) written in Go. The
project current contains a working CPU implementation and a number of tests to
ensure correctness.

The virtual machine is implemented as an object so that several can be
instantiated and run in separate goroutines.

The virtual machine is cycle-time accurate. It is currently running at 1 GHz,
so the tests complete in a reasonable timeframe.
