package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nickwells/col.mod/v3/col"
	"github.com/nickwells/col.mod/v3/col/colfmt"
	"github.com/nickwells/twrap.mod/twrap"
)

const (
	ColLevel        = "level"
	ColName         = "name"
	ColUseCount     = "use-count"
	ColUsedBy       = "used-by"
	ColUsesCountInt = "uses-count-int"
	ColUsesCountExt = "uses-count-ext"
	ColPackages     = "packages"
	ColPkgLines     = "lines-of-code"
)

var columnsToShow = map[string]bool{
	ColLevel:    true,
	ColName:     true,
	ColUseCount: true,
}

var (
	hideDupLevels bool
	canSkipCols   = true
	showIntro     = true
	showHeader    = true

	sortBy = ColLevel

	modFilter map[string]bool
)

var helpTxt = "The level value indicates that the module requires modules" +
	" having lower level values and does not require any modules having" +
	" a higher level." +
	"\n\n" +
	"The use count shows how many other modules require this module. A" +
	" high use count means if you change this module, you'll have to" +
	" update the go.mod file of many other modules." +
	"\n\n" +
	"The used-by columns shows which other modules require this module." +
	"\n\n" +
	"The uses count (internal) indicates how many other modules from" +
	" this collection this module requires." +
	"\n\n" +
	"The uses count (external) indicates how many modules from outside" +
	" this collection this module requires." +
	"\n\n" +
	"The packages count shows how many directories with Go source code" +
	" there are in the module. This may be 'main' packages (generating" +
	" executable binaries)." +
	"\n\n" +
	"The package lines of code shows how many non-test lines there are" +
	" in the packages." +
	"\n\n" +
	"This allows you to make judgements about changes you are making." +
	" For instance, if you are changing a module at level 3," +
	" you might have to make changes to other modules with" +
	" higher levels (4 or greater) but you will not have to" +
	" make any changes to modules with levels 3 or less." +
	" If you make changes to a module with a zero use count" +
	" you know that no other modules will be affected." +
	" Alternatively, if you change a module with a high use count" +
	" then many other modules will be impacted."

// printReportIntro prints the report introduction
func printReportIntro(w io.Writer, n uint64) {
	if n != 0 {
		return
	}
	twc := twrap.NewTWConfOrPanic(twrap.SetWriter(w))
	twc.Wrap("This shows how the modules relate to one another.\n\n"+helpTxt, 0)
}

// makeHeader constructs the header and returns it with an error. If the
// error is not nii the header is invalid
func makeHeader() (*col.Header, error) {
	hdrOpts := make([]col.HdrOptionFunc, 0)
	if !showHeader {
		hdrOpts = append(hdrOpts, col.HdrOptDontPrint)
	}
	if showIntro {
		hdrOpts = append(hdrOpts,
			col.HdrOptPreHdrFunc(printReportIntro),
		)
	}
	return col.NewHeader(hdrOpts...)
}

// makeReport constructs the report and returns it with an error. If the
// error is not nii the report is invalid
func (modules ModMap) makeReport(h *col.Header) *col.Report {
	cols := make([]*col.Col, 0, len(columnsToShow))

	if columnsToShow[ColLevel] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Level"))
	}
	cols = append(cols,
		col.New(colfmt.String{W: modules.findMaxNameLen()}, "Module name"))
	if columnsToShow[ColUseCount] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Used By"))
	}
	if columnsToShow[ColUsesCountInt] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Uses (int)"))
	}
	if columnsToShow[ColUsesCountExt] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Uses (ext)"))
	}
	if columnsToShow[ColPackages] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Packages"))
	}
	if columnsToShow[ColPkgLines] {
		cols = append(cols, col.New(colfmt.Int{W: 7}, "Package", "LoC"))
	}
	if columnsToShow[ColUsedBy] {
		cols = append(cols, col.New(colfmt.String{}, "Used By"))
	}

	if len(cols) == 1 {
		return col.NewReport(h, os.Stdout, cols[0])
	}
	return col.NewReport(h, os.Stdout, cols[0], cols[1:]...)
}

// addLevelCol adds the level column value to the colVals and returns the new
// colVals
func addLevelCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColLevel] {
		colVals = append(colVals, mi.Level)
	}
	return colVals
}

// addUseCountCol adds the use count column value to the colVals and returns
// the new colVals
func addUseCountCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColUseCount] {
		colVals = append(colVals, len(mi.ReqdBy))
	}
	return colVals
}

// addUsedByCol adds the use count column value to the colVals and returns
// the new colVals
func addUsedByCol(mi *ModInfo, colVals []any, i int) []any {
	if columnsToShow[ColUsedBy] {
		val := ""
		if len(mi.ReqdBy) > 0 {
			val = mi.ReqdBy[i].Name
		}
		colVals = append(colVals, val)
	}
	return colVals
}

// addUsesCountIntCol adds the uses count (internal) column value to the
// colVals and returns the new colVals
func addUsesCountIntCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColUsesCountInt] {
		colVals = append(colVals, mi.ReqCountInternal)
	}
	return colVals
}

// addUsesCountExtCol adds the uses count (external) column value to the
// colVals and returns the new colVals
func addUsesCountExtCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColUsesCountExt] {
		colVals = append(colVals, mi.ReqCountExternal)
	}
	return colVals
}

// addPackagesCol adds the number of packages provided column value to the
// colVals and returns the new colVals
func addPackagesCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColPackages] {
		colVals = append(colVals, len(mi.Packages))
	}
	return colVals
}

// addPackagesLoCCol adds the number of lines of package code column value to
// the colVals and returns the new colVals. Note that only non-test files are
// counted.
func addPackagesLoCCol(mi *ModInfo, colVals []any) []any {
	if columnsToShow[ColPackages] {
		linesOfCode := 0
		for _, pkg := range mi.Packages {
			for _, gi := range pkg.Files {
				linesOfCode += gi.LineCount
			}
		}
		colVals = append(colVals, linesOfCode)
	}
	return colVals
}

// addReqsToFilters will add all the ReqdBy entries for the module into the
// filter map.
func addReqsToFilters(mi *ModInfo) {
	for _, rb := range mi.ReqdBy {
		modFilter[rb.Name] = true
	}
}

// skipModInfo returns true if the module info record should be skipped. It
// will be skippped if:
//
// The location has not been filled in (its an external module).
//
// There is a module filter and the name does not match an entry in the
// filters map.
func skipModInfo(mi *ModInfo) bool {
	if mi.Loc == nil {
		return true
	}

	if len(modFilter) > 0 && !modFilter[mi.Name] {
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
func printModInfo(rpt *col.Report, mi *ModInfo, lastLevel int) error {
	vals := make([]any, 0, len(columnsToShow))
	var skipCount uint
	if lastLevel == mi.Level &&
		hideDupLevels &&
		columnsToShow[ColLevel] &&
		canSkipCols {
		skipCount = 1
	} else {
		vals = addLevelCol(mi, vals)
	}
	vals = append(vals, mi.Name)
	vals = addUseCountCol(mi, vals)
	vals = addUsesCountIntCol(mi, vals)
	vals = addUsesCountExtCol(mi, vals)
	vals = addPackagesCol(mi, vals)
	vals = addPackagesLoCCol(mi, vals)
	var skipCountExtras uint
	if canSkipCols {
		skipCountExtras = uint(len(vals))
	}

	err := rpt.PrintRowSkipCols(skipCount, addUsedByCol(mi, vals, 0)...)
	if err != nil {
		return err
	}
	return reportExtraUsedByValues(rpt, skipCount+skipCountExtras, vals, mi)
}

// reportExtraUsedByValues reports any additional UsedBy module names
func reportExtraUsedByValues(rpt *col.Report, skip uint,
	vals []any, mi *ModInfo,
) error {
	if !columnsToShow[ColUsedBy] {
		return nil
	}

	if canSkipCols {
		vals = vals[:0]
	}
	for i := 1; i < len(mi.ReqdBy); i++ {
		err := rpt.PrintRowSkipCols(skip, addUsedByCol(mi, vals, i)...)
		if err != nil {
			return err
		}
	}

	return nil
}

// reportModuleInfo prints the module information
func (modules ModMap) reportModuleInfo() {
	h, err := makeHeader()
	if err != nil {
		fmt.Println("Couldn't make the report header:", err)
		return
	}
	rpt := modules.makeReport(h)

	lastLevel := -1
	for _, mi := range modules.makeModInfoSlice(sortBy) {
		if skipModInfo(mi) {
			continue
		}

		err = printModInfo(rpt, mi, lastLevel)

		if err != nil {
			fmt.Println("Error found while printing the report:", err)
			break
		}
		lastLevel = mi.Level
	}
}
