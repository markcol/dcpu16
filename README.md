DCPU-16 Vitual Machine tools written in Go
==========================================
[![Build Status](https://travis-ci.org/markcol/dcpu16.png?branch=master)](https://travis-ci.org/markcol/dcpu16)

This is a set of tools for creating, test and running a virtual
machine based on the [DCPU-16 specification](http://dcpu.com/dcpu-16/)
written in Go. The project contains:

   * A cycle-accurate CPU
   * Assembler
   * Disassmbler
   * Virtual machine

Each component has a number of tests to ensure functional correctness.

The virtual machine is overclocked to 1 GHz by default only so that
tests complete in a reasonable amount of time.
