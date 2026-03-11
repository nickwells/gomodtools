package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/twrap.mod/twrap"
)

// makeReportIntroFunc returns a function that can be supplied when
// constructing a report header and will be called before the header is
// printed.
func makeReportIntroFunc(prog *prog) col.PreHdrFunc {
	const colNameIndent = 4

	var maxColNameLen = 0

	for _, c := range prog.columnsToShow {
		maxColNameLen = max(maxColNameLen, len(c))
	}

	return func(w io.Writer, i int64) {
		if i != 0 {
			fmt.Fprintln(w)
			return
		}

		twc := twrap.NewTWConfOrPanic(twrap.SetWriter(w))

		twc.Wrap("This gives information about a collection of modules"+
			" and how they relate to one another."+
			" The information in this report can be interpreted as follows.",
			0)

		for _, c := range prog.columnsToShow {
			twc.Println()
			twc.WrapPrefixed(fmt.Sprintf("%-*s ", maxColNameLen, c),
				columnDescription[c],
				colNameIndent)
		}
		twc.Println()
	}
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
			col.HdrOptPreHdrFunc(makeReportIntroFunc(prog)),
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
