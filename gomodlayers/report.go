package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/twrap.mod/twrap"
)

// printReportIntro prints the report introduction
func printReportIntro(w io.Writer, n int64) {
	if n != 0 {
		return
	}

	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(w))

	twc.Wrap("This shows how the modules relate to one another."+
		"\n\n"+
		"The level value indicates that the module requires modules"+
		" having lower level values and does not require any modules having"+
		" a higher level."+
		"\n\n"+
		"The use count shows how many other modules require this module. A"+
		" high use count means if you change this module, you'll have to"+
		" update the go.mod file of many other modules."+
		"\n\n"+
		"The used-by columns shows which other modules require this module."+
		"\n\n"+
		"The uses count (internal) indicates how many other modules from"+
		" this collection this module requires."+
		"\n\n"+
		"The uses count (external) indicates how many modules from outside"+
		" this collection this module requires."+
		"\n\n"+
		"The packages count shows how many directories with Go source code"+
		" there are in the module. This may be 'main' packages (generating"+
		" executable binaries)."+
		"\n\n"+
		"The package lines of code shows how many non-test lines there are"+
		" in the packages."+
		"\n\n"+
		"This allows you to make judgements about changes you are making."+
		" For instance, if you are changing a module at level 3,"+
		" you might have to make changes to other modules with"+
		" higher levels (4 or greater) but you will not have to"+
		" make any changes to modules with levels 3 or less."+
		" If you make changes to a module with a zero use count"+
		" you know that no other modules will be affected."+
		" Alternatively, if you change a module with a high use count"+
		" then many other modules will be impacted.",
		0)
}

// makeHeader constructs the header and returns it with an error. If the
// error is not nii the header is invalid
func (prog *prog) makeHeader() (*col.Header, error) {
	hdrOpts := make([]col.HdrOptionFunc, 0)

	if !prog.showHeader {
		hdrOpts = append(hdrOpts, col.HdrOptDontPrint)
	}

	if prog.showIntro {
		hdrOpts = append(hdrOpts,
			col.HdrOptPreHdrFunc(printReportIntro),
		)
	}

	if prog.headerRepeat > 0 {
		hdrOpts = append(hdrOpts,
			col.HdrOptRepeat(int64(prog.headerRepeat)),
		)
	}

	return col.NewHeader(hdrOpts...)
}

// makeReport constructs the report and returns it with an error. If the
// error is not nii the report is invalid
func (modules modMap) makeReport(h *col.Header, prog *prog) *col.Report {
	cols := make([]*col.Col, 0, len(prog.columnsToShow))

	for _, c := range prog.columnsToShow {
		cols = append(cols, reportColumns[c](prog, modules))
	}

	if len(cols) == 1 {
		return col.NewReportOrPanic(h, os.Stdout, cols[0])
	}

	return col.NewReportOrPanic(h, os.Stdout, cols[0], cols[1:]...)
}

// addReqsToFilters will add all the ReqdBy entries for the module into the
// filter map.
func (prog *prog) addReqsToFilters(mi *modInfo) {
	for _, rb := range mi.ReqdBy {
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

// printModInfo gathers the values to be printed and then prints the row. It
// calculates the columns to be skipped (unless canSkipCols is set to false):
//
// Firstly the level column is skipped if it is the same as the previous
// level, the hideDupLevels flag is set and module levels are being shown.
//
// For the first row of each module this is all that is skipped but for
// subsequent rows all the columns up to the UsedBy column are skipped
func (prog *prog) printModInfo(rpt *col.Report, mi *modInfo) error {
	vals := make([]any, 0, len(prog.columnsToShow)+1)
	for _, c := range prog.columnsToShow {
		vals = append(vals, columnVals[c](prog, mi))
	}

	return rpt.PrintRow(vals...)
}

// reportModuleInfo prints the module information
func (modules modMap) reportModuleInfo(prog *prog) {
	h, err := prog.makeHeader()
	if err != nil {
		fmt.Println("Couldn't make the report header:", err)
		return
	}

	rpt := modules.makeReport(h, prog)

	for i, mi := range modules.makeModInfoSlice(prog.sortBy) {
		if prog.skipModInfo(mi) {
			continue
		}

		err = prog.printModInfo(rpt, mi)
		if err != nil {
			fmt.Printf("Couldn't print line %d of the report (module: %q): %s",
				i, mi.Name, err)

			return
		}
	}
}
