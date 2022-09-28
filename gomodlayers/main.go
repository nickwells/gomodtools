package main

// gomodlayers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nickwells/col.mod/v3/col"
	"github.com/nickwells/col.mod/v3/col/colfmt"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Thu Mar 28 12:13:29 2019

const (
	ColLevel        = "level"
	ColName         = "name"
	ColUseCount     = "use-count"
	ColUsedBy       = "used-by"
	ColUsesCountInt = "uses-count-int"
	ColUsesCountExt = "uses-count-ext"
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

// ModInfo records information gleaned from the go.mod files
type ModInfo struct {
	Loc              *location.L
	Name             string
	Reqs             []*ModInfo
	ReqCountInternal int
	ReqCountExternal int
	Level            int
	ReqdBy           []*ModInfo
}

var modules = map[string]*ModInfo{}

var helpTxt = "The level value indicates that the module requires modules" +
	" having lower level values and does not require any modules having" +
	" a higher level." +
	"\n\n" +
	"The use count shows how many other modules require this module. A" +
	" high use count means if you change this module, you'll have to" +
	" update the go.mod file of many other modules." +
	"\n\n" +
	"The uses count (internal) indicates how many other modules from" +
	" this collection this module requires." +
	"\n\n" +
	"The uses count (external) indicates how many modules from outside" +
	" this collection this module requires." +
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

func main() {
	ps := paramset.NewOrDie(
		versionparams.AddParams,
		addParams,
		addExamples,
		SetGlobalConfigFile,
		SetConfigFile,
		param.SetProgramDescription("This will take a list of go.mod"+
			" files (or directories) as trailing arguments"+
			" (after '"+param.DfltTerminalParam+"'), parse them and print"+
			" a report. The report will show how they relate to one"+
			" another with regards to dependencies and can print them in"+
			" such an order that an earlier module does not depend on any"+
			" subsequent module."+
			"\n\n"+
			"By default any report will be preceded with a description of"+
			" what the various columns mean."+
			"\n\n"+
			"If one of the trailiing arguments does not end with '/go.mod'"+
			" then it is taken as a directory name and the missing"+
			" filename is automatically appended."),
	)

	ps.Parse()

	parseAllGoModFiles(ps.Remainder())
	calcLevels()
	calcReqCount()
	expandModFilters()
	reportModuleInfo()
}

// parseAllGoModFiles will process the list of filenames, opening each one in
// turn and populating the moduleInfo map. If any filename doesn't end with
// go.mod then that is added to the end of the path before further processing
func parseAllGoModFiles(goModFilenames []string) {
	const goMod = "go.mod"
	for _, fname := range goModFilenames {
		if !strings.HasSuffix(fname, goMod) {
			fname = filepath.Join(fname, goMod)
		}

		f, err := os.Open(fname)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		parseGoModFile(f, location.New(fname))
	}
}

// parseGoModFile parses the supplied file and uses the information found to
// populate the map of moduleInfo
func parseGoModFile(f io.Reader, loc *location.L) {
	spacePattern := regexp.MustCompile("[ \t]+")

	scanner := bufio.NewScanner(f)
	var mi *ModInfo
	var inReqBlock bool
	for scanner.Scan() {
		loc.Incr()
		line := scanner.Text()
		switch line {
		case "":
			continue
		case "require (":
			inReqBlock = true
			continue
		case ")":
			inReqBlock = false
			continue
		}

		parts := spacePattern.Split(line, -1)
		switch parts[0] {
		case "module":
			loc.SetContent(line)
			mi = getModuleInfo(parts, loc)
			if mi == nil {
				break
			}
		case "":
			if inReqBlock {
				loc.SetContent(line)
				populateRequirements(mi, parts, loc)
			}
		case "require":
			loc.SetContent(line)
			populateRequirements(mi, parts, loc)
		}
	}
	initLevel(mi, loc)
}

// initLevel sets the initial level of the module. It reports an error if
// there is no module (if the pointer is nil)
func initLevel(mi *ModInfo, loc *location.L) {
	if mi == nil {
		fmt.Fprintf(os.Stderr,
			"Error: there is no module defined in file: %s\n", loc.Source())
		return
	}

	if len(mi.Reqs) > 0 {
		mi.Level = 1
	}
}

// populateRequirements expects to be passed a non-nil ModInfo and the parts
// of a require line. It will find the corresponding module for the required
// module and record that as a requirement of the module and also record that
// this module requires the other module. If there is a problem it will
// report it and return false, otherwise it returns true.
func populateRequirements(mi *ModInfo, parts []string, loc *location.L) bool {
	if mi == nil {
		fmt.Fprintf(os.Stderr, "Error: no module is defined at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : the module should be known\n")
		fmt.Fprintf(os.Stderr, "     : before requirements are stated\n")
		return false
	}

	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "Error: there is no module name at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : too few parts\n")
		fmt.Fprintf(os.Stderr, "     : a required module name was expected\n")
		return false
	}

	r := parts[1]
	reqdMI, ok := modules[r]
	if !ok {
		reqdMI = &ModInfo{
			Name: r,
		}
		modules[r] = reqdMI
	}
	reqdMI.ReqdBy = append(reqdMI.ReqdBy, mi)
	mi.Reqs = append(mi.Reqs, reqdMI)

	return true
}

// getModuleInfo gets the module info for the named module. It will return
// nil if the module has already been defined (if it's seen a line starting
// with the word "module")
func getModuleInfo(parts []string, loc *location.L) *ModInfo {
	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "Error: there is no module name at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : too few parts\n")
		fmt.Fprintf(os.Stderr, "     : a module name was expected\n")
		return nil
	}

	modName := parts[1]

	mi, ok := modules[modName]
	if !ok {
		mi = &ModInfo{
			Name: modName,
			Loc:  loc,
		}
		modules[modName] = mi
		return mi
	}

	if mi.Loc == nil {
		mi.Loc = loc
		return mi
	}

	fmt.Fprintf(os.Stderr, "Error: module %s has been declared before", modName)
	fmt.Fprintf(os.Stderr, "     : firstly at %s\n", mi.Loc)
	fmt.Fprintf(os.Stderr, "     :     now at %s\n", loc)

	return nil
}

// calcLevels will repeatedly go over the modules resetting the level to be
// one greater than that of the highest level module which it requires. It
// keeps on doing this until it has made no further changes.
func calcLevels() {
	levelChange := true
	for levelChange {
		levelChange = false
		for _, mi := range modules {
			for _, rmi := range mi.Reqs {
				if rmi.Level >= mi.Level {
					mi.Level = rmi.Level + 1
					levelChange = true
				}
			}
		}
	}
}

// calcReqCount will calculate the number of internal and external
// requirements for each module. If a required module has no location set
// then it is taken to be an external requireement.
func calcReqCount() {
	for _, mi := range modules {
		mi.ReqCountInternal = 0
		mi.ReqCountExternal = 0

		for _, rmi := range mi.Reqs {
			if rmi.Loc == nil {
				mi.ReqCountExternal++
			} else {
				mi.ReqCountInternal++
			}
		}
	}
}

// findMaxNameLen returns the length of the longest module name
func findMaxNameLen() int {
	max := 0
	for _, mi := range modules {
		if len(mi.Name) > max {
			max = len(mi.Name)
		}
	}
	return max
}

// makeModInfoSlice returns the modules map as a slice of ModInfo
// pointers. The slice will be sorted according to the value of the sort
// parameter
func makeModInfoSlice(order string) []*ModInfo {
	ms := make([]*ModInfo, 0, len(modules))
	for _, mi := range modules {
		ms = append(ms, mi)
	}

	switch order {
	case ColLevel:
		sort.Slice(ms, func(i, j int) bool { return lessByLevel(ms, i, j) })
	case ColName:
		sort.Slice(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })
	case ColUseCount:
		sort.Slice(ms, func(i, j int) bool { return lessByUseCount(ms, i, j) })
	case ColUsesCountInt:
		sort.Slice(ms,
			func(i, j int) bool { return lessByReqCountInt(ms, i, j) })
	case ColUsesCountExt:
		sort.Slice(ms,
			func(i, j int) bool { return lessByReqCountExt(ms, i, j) })
	}
	return ms
}

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
func makeReport(h *col.Header) *col.Report {
	cols := make([]*col.Col, 0, len(columnsToShow))

	if columnsToShow[ColLevel] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Level"))
	}
	cols = append(cols,
		col.New(colfmt.String{W: findMaxNameLen()}, "Module name"))
	if columnsToShow[ColUseCount] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Used By"))
	}
	if columnsToShow[ColUsesCountInt] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Uses (int)"))
	}
	if columnsToShow[ColUsesCountExt] {
		cols = append(cols, col.New(colfmt.Int{W: 3}, "Count", "Uses (ext)"))
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

// addReqsToFilters will add all the ReqdBy entries for the module into the
// filter map.
func addReqsToFilters(mi *ModInfo) {
	for _, rb := range mi.ReqdBy {
		modFilter[rb.Name] = true
	}
}

// expandModFilters takes the initial set of modFilters and adds all the
// other modules that it is required by.
func expandModFilters() {
	if len(modFilter) == 0 {
		return
	}

	for _, mi := range makeModInfoSlice(ColLevel) {
		if modFilter[mi.Name] {
			addReqsToFilters(mi)
		}
	}
}

// skipModInfo returns true if the module info record should be skipped. It
// will be skippped if:
//
// The location has not been filled in (if its an external module).
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
func reportModuleInfo() {
	h, err := makeHeader()
	if err != nil {
		fmt.Println("Couldn't make the report header:", err)
		return
	}
	rpt := makeReport(h)

	lastLevel := -1
	for _, mi := range makeModInfoSlice(sortBy) {
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
