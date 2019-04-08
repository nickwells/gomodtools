// gomodlayers
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/nickwells/col.mod/col"
	"github.com/nickwells/col.mod/col/colfmt"
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v2/param"
	"github.com/nickwells/param.mod/v2/param/paramset"
	"github.com/nickwells/param.mod/v2/param/psetter"
	"github.com/nickwells/twrap.mod/twrap"
	"github.com/nickwells/xdg.mod/xdg"
)

// Created: Thu Mar 28 12:13:29 2019

const (
	ColLevel        = "level"
	ColName         = "name"
	ColUseCount     = "use-count"
	ColUsesCountInt = "uses-count-int"
	ColUsesCountExt = "uses-count-ext"
)

var columnsToShow = map[string]bool{
	ColLevel:    true,
	ColName:     true,
	ColUseCount: true,
}

var hideDupLevels bool
var showIntro = true
var showHeader = true
var sortBy = ColLevel

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
	" a higher level.\n\n" +
	"The use count indicates how many other modules require this" +
	" module.\n\n" +
	"The uses count (internal) indicates how many other modules from" +
	" this collection this module requires.\n\n" +
	"The uses count (external) indicates how many modules from outside" +
	" this collection this module requires.\n\n" +
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
	ps, err := paramset.New(addParams,
		param.SetProgramDescription("This will parse the provided"+
			" go.mod files and will print a report of where they sit"+
			" in relation to one another.\n\n"+helpTxt),
	)
	if err != nil {
		log.Fatal("Couldn't construct the parameter set: ", err)
	}
	ps.AddConfigFile(
		filepath.Join(xdg.ConfigHome(), "golem", "gomodtools.cfg"),
		filecheck.Optional)

	ps.Parse()

	parseAllGoModFiles(ps.Remainder())
	calcLevels()
	calcReqCount()
	reportModuleInfo()
}

// parseAllGoModFiles will process the list of filenames, opening each one in
// turn and populating the moduleInfo map
func parseAllGoModFiles(goModFilenames []string) {
	for _, fname := range goModFilenames {
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
// this module requires the other module. If a problem it will report it and
// return false, otherwise it returns true.
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

// lessByLevel returns true or false according to the levels of the ModInfo
// entries. It will use the module name to resolve ties.
func lessByLevel(ms []*ModInfo, i, j int) bool {
	if ms[i].Level < ms[j].Level {
		return true
	}
	if ms[i].Level > ms[j].Level {
		return false
	}
	return ms[i].Name < ms[j].Name
}

// lessByUseCount returns true or false according to the UseCounts of the
// ModInfo entries. It will use the module name to resolve ties.
func lessByUseCount(ms []*ModInfo, i, j int) bool {
	if len(ms[i].ReqdBy) < len(ms[j].ReqdBy) {
		return true
	}
	if len(ms[i].ReqdBy) > len(ms[j].ReqdBy) {
		return false
	}
	return ms[i].Name < ms[j].Name
}

// lessByReqCountInt returns true or false according to the internal
// requirement count of the ModInfo entries. It will use the module name to
// resolve ties.
func lessByReqCountInt(ms []*ModInfo, i, j int) bool {
	if ms[i].ReqCountInternal < ms[j].ReqCountInternal {
		return true
	}
	if ms[i].ReqCountInternal > ms[j].ReqCountInternal {
		return false
	}
	return ms[i].Name < ms[j].Name
}

// lessByReqCountExt returns true or false according to the external
// requirement count of the ModInfo entries. It will use the module name to
// resolve ties.
func lessByReqCountExt(ms []*ModInfo, i, j int) bool {
	if ms[i].ReqCountExternal < ms[j].ReqCountExternal {
		return true
	}
	if ms[i].ReqCountExternal > ms[j].ReqCountExternal {
		return false
	}
	return ms[i].Name < ms[j].Name
}

// makeModInfoSlice returns the modules map as a slice of ModInfo
// pointers. The slice will be sorted according to the value of the sort
// parameter
func makeModInfoSlice() []*ModInfo {
	ms := make([]*ModInfo, 0, len(modules))
	for _, mi := range modules {
		ms = append(ms, mi)
	}

	if sortBy == ColLevel {
		sort.Slice(ms, func(i, j int) bool { return lessByLevel(ms, i, j) })
	} else if sortBy == ColName {
		sort.Slice(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })
	} else if sortBy == ColUseCount {
		sort.Slice(ms, func(i, j int) bool { return lessByUseCount(ms, i, j) })
	} else if sortBy == ColUsesCountInt {
		sort.Slice(ms, func(i, j int) bool { return lessByReqCountInt(ms, i, j) })
	} else if sortBy == ColUsesCountExt {
		sort.Slice(ms, func(i, j int) bool { return lessByReqCountExt(ms, i, j) })
	}
	return ms
}

// printReportIntro prints the report introduction
func printReportIntro(w io.Writer, n uint64) {
	if n != 0 {
		return
	}
	twc, err := twrap.NewTWConf(twrap.TWConfOptSetWriter(w))
	if err != nil {
		fmt.Fprintf(w, "Couldn't make the text wrap configuration: %s", err)
		return
	}
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
func makeReport(h *col.Header) (*col.Report, error) {
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
	return col.NewReport(h, os.Stdout, cols...)
}

// addLevelCol adds the level column value to the colVals and returns the new
// colVals
func addLevelCol(mi *ModInfo, colVals []interface{}) []interface{} {
	if columnsToShow[ColLevel] {
		colVals = append(colVals, mi.Level)
	}
	return colVals
}

// addUseCountCol adds the use count column value to the colVals and returns
// the new colVals
func addUseCountCol(mi *ModInfo, colVals []interface{}) []interface{} {
	if columnsToShow[ColUseCount] {
		colVals = append(colVals, len(mi.ReqdBy))
	}
	return colVals
}

// addUsesCountIntCol adds the uses count (internal) column value to the
// colVals and returns the new colVals
func addUsesCountIntCol(mi *ModInfo, colVals []interface{}) []interface{} {
	if columnsToShow[ColUsesCountInt] {
		colVals = append(colVals, mi.ReqCountInternal)
	}
	return colVals
}

// addUsesCountExtCol adds the uses count (external) column value to the
// colVals and returns the new colVals
func addUsesCountExtCol(mi *ModInfo, colVals []interface{}) []interface{} {
	if columnsToShow[ColUsesCountExt] {
		colVals = append(colVals, mi.ReqCountExternal)
	}
	return colVals
}

// reportModuleInfo prints the module information
func reportModuleInfo() {
	h, err := makeHeader()
	if err != nil {
		fmt.Println("Error found while constructing the report header:", err)
		return
	}
	rpt, err := makeReport(h)
	if err != nil {
		fmt.Println("Error found while constructing the report:", err)
		return
	}

	lastLevel := -1
	for _, mi := range makeModInfoSlice() {
		if mi.Loc == nil {
			continue
		}
		colVals := make([]interface{}, 0, len(columnsToShow))
		if lastLevel == mi.Level && hideDupLevels && columnsToShow[ColLevel] {
			colVals = append(colVals, mi.Name)
			colVals = addUseCountCol(mi, colVals)
			colVals = addUsesCountIntCol(mi, colVals)
			colVals = addUsesCountExtCol(mi, colVals)

			err = rpt.PrintRowSkipCols(1, colVals...)
		} else {
			colVals = addLevelCol(mi, colVals)
			colVals = append(colVals, mi.Name)
			colVals = addUseCountCol(mi, colVals)
			colVals = addUsesCountIntCol(mi, colVals)
			colVals = addUsesCountExtCol(mi, colVals)

			err = rpt.PrintRow(colVals...)
		}
		if err != nil {
			fmt.Println("Error found while printing the report:", err)
			break
		}
		lastLevel = mi.Level
	}
}

// addParams will add parameters to the passed PSet
func addParams(ps *param.PSet) error {
	ps.Add("hide-header", psetter.Bool{Value: &showHeader, Invert: true},
		"suppress the printing of the header",
	)
	ps.Add("hide-dup-levels", psetter.Bool{Value: &hideDupLevels},
		"suppress the printing of levels where the lavel value"+
			" is the same as on the previous line",
	)
	ps.Add("hide-intro", psetter.Bool{Value: &showIntro, Invert: true},
		"suppress the printing of the introductory text"+
			" explaining the meaning of the report",
	)

	ps.Add("sort-order",
		psetter.Enum{
			Value: &sortBy,
			AllowedVals: psetter.AValMap{
				ColLevel:    "in level order (lowest first)",
				ColName:     "in name order",
				ColUseCount: "in order of how heavily used the module is",
				ColUsesCountInt: "in order of how much use the module makes" +
					" of other modules in the collection",
				ColUsesCountExt: "in order of how much use the module makes" +
					" of modules not in the collection",
			}},
		"what order should the modules be sorted when reporting",
		param.AltName("sort-by"),
	)

	ps.Add("show-cols",
		psetter.EnumMap{
			Value: &columnsToShow,
			AllowedVals: psetter.AValMap{
				ColLevel:    "where the module lies in the dependency order",
				ColUseCount: "how heavily used the module is",
				ColUsesCountInt: "how much use the module makes" +
					" of other modules in the collection",
				ColUsesCountExt: "how much use the module makes" +
					" of modules not in the collection",
			},
			AllowHiddenMapEntries: true,
		},
		"what columns should be shown",
	)

	err := ps.SetRemHandler(param.NullRemHandler{}) // allow trailing arguments
	if err != nil {
		return err
	}

	return nil
}
