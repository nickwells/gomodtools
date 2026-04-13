package main

import (
	"fmt"
	"os"
)

// addReqsToFilters will add all the ReqdBy entries for the module into the
// filter map.
func (prog *prog) addReqsToFilters(mi *modInfo) {
	for _, rb := range mi.ReqdByDirectly {
		prog.modFilter[rb.Name] = true
	}
}

// skipModInfo returns true if the module info record should be skipped. It
// will be skippped if:
//
// The location has not been filled in (its an external module).
//
// There is a module filter and the name does not match an entry in the
// filters map.
func (prog *prog) skipModInfo(mi *modInfo) bool {
	if mi.Loc == nil {
		return true
	}

	if len(prog.modFilter) > 0 && !prog.modFilter[mi.Name] {
		return true
	}

	return false
}

// reportModuleInfo prints the module information
func (modules modMap) reportModuleInfo(prog *prog) {
	reporter, err := prog.cols.MakeReport(prog,
		os.Stdout,
		prog.columnsToShow,
		prog.headerOptFuncs()...)
	if err != nil {
		fmt.Println("Couldn't make the report:", err)
		return
	}

	prog.populateModInfo(modules)

	if err = reporter.Print(prog.mInfo, prog.makeSortCols()); err != nil {
		fmt.Println("Couldn't print the report:", err)

		return
	}
}
