package main

// gomodlayers

import (
	"os"
)

// Created: Thu Mar 28 12:13:29 2019

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	prog.run()
	os.Exit(prog.exitStatus)
}
