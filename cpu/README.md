DCPU-16 cycle-accurate CPU
==========================

This is an implementation of a CPU capable of executing instructions
as defined by the [DCPU-16 specification](http://dcpu.com/dcpu-16/)
written in Go.

The virtual machine is implemented as an object so that several can be
instantiated and run in separate goroutines.

The virtual machine is cycle-time accurate, and runs at 1 GHz,
so the tests complete in a reasonable timeframe.
