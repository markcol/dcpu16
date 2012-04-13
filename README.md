DCPU-16 Vitual Machine tools written in Go
==========================================

This is an implementation of the
[DCPU-16 specification](http://0x10c.com/doc/dcpu-16.txt"") written in Go. The
project current contains a working CPU implementation and a number of tests to
ensure correctness.

The virtual machine is implemented as an object so that several can be
instantiated and run in separate goroutines. 

