package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/rptmaker"
	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/twrap.mod/twrap"
)

// sortCol is a type alias for the TaggedEnum
type sortCol = psetter.TaggedValue[rptmaker.ColID, rptmaker.SortWay]

type OutputStyle string

const (
	styleReport  = "report"
	styleDotFile = "dotfile"
)

// prog holds program parameters, intermediate results and status
type prog struct {
	exitStatus int

	hideDupLevels bool
	showIntro     bool
	showHeader    bool

	sortBy []sortCol

	modFilter     map[string]bool
	partialFilter map[string]bool
	backFilter    map[string]bool
	hideModules   map[string]bool

	columnsToShow []rptmaker.ColID

	moduleFiles []string
	mm          modMap
	mInfo       []*modInfo

	maxNameLen int

	reportDigits int
	headerRepeat int

	output OutputStyle

	cols *rptmaker.Cols[*prog, *modInfo]

	dotFileDir string

	stripPrefix string
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	const dfltDigitsToShow = 5

	prog := &prog{
		showIntro:  true,
		showHeader: true,

		sortBy:        []sortCol{{Value: ColLevel}, {Value: ColName}},
		columnsToShow: []rptmaker.ColID{ColLevel, ColName, ColUseCountTotal},

		modFilter:     map[string]bool{},
		partialFilter: map[string]bool{},
		backFilter:    map[string]bool{},

		mm: modMap{},

		reportDigits: dfltDigitsToShow,

		output: styleReport,
	}

	prog.cols = prog.populateCols()

	return prog
}

// setExitStatus sets the exit status to the new value. It will not do this
// if the exit status has already been set to a non-zero value.
func (prog *prog) setExitStatus(es int) {
	if prog.exitStatus == 0 {
		prog.exitStatus = es
	}
}

// run generates the module report
func (prog *prog) run() {
	if errMap := prog.mm.populate(prog.moduleFiles); errMap.HasErrors() {
		errMap.Report(os.Stderr, "")
		prog.setExitStatus(1)

		return
	}

	prog.maxNameLen = prog.mm.findMaxNameLen()

	prog.mm.calcLevels()
	prog.mm.calcReqCount()

	prog.expandModFilters()
	prog.populateModInfo()

	switch prog.output {
	case styleReport:
		prog.reportModuleInfo()
	case styleDotFile:
		prog.makeDotfile()
	}
}

// applyBackFilters takes all the back filters and adds their
// requirements to the set of filters
func (prog *prog) applyBackFilters() {
	for _, mi := range prog.mm {
		if prog.backFilter[mi.Name] {
			prog.modFilter[mi.Name] = true

			for _, dr := range mi.DirectReqs {
				prog.modFilter[dr.Name] = true
			}

			for _, ir := range mi.IndirectReqs {
				prog.modFilter[ir.Name] = true
			}
		}
	}
}

// applyForwardFilters takes all the filters and adds those modules that
// require them (directly) to the set of filters
func (prog *prog) applyForwardFilters() {
	for _, mi := range prog.mm {
		if prog.modFilter[mi.Name] {
			for _, rb := range mi.ReqdByDirectly {
				prog.modFilter[rb.Name] = true
			}
		} else {
			if prog.matchPartialFilters(mi.Name) {
				prog.modFilter[mi.Name] = true
				for _, rb := range mi.ReqdByDirectly {
					prog.modFilter[rb.Name] = true
				}
			}
		}
	}
}

// expandModFilters takes the initial set of modFilters and adds all the
// other modules that it is required by.
func (prog *prog) expandModFilters() {
	if len(prog.modFilter) == 0 &&
		len(prog.partialFilter) == 0 &&
		len(prog.backFilter) == 0 {
		return
	}

	prog.applyForwardFilters()
	prog.applyBackFilters()
}

// makeSortCols converts the prog.sortBy slice into a slice of
// [rptmaker.SortColumns].
func (prog *prog) makeSortCols() []rptmaker.SortColumn {
	sortCols := make([]rptmaker.SortColumn, 0, len(prog.sortBy))

	for _, sc := range prog.sortBy {
		sortCols = append(sortCols, rptmaker.MakeSortColumn(sc.Value, sc.Tags))
	}

	return sortCols
}

var versionRE = regexp.MustCompile(`v[1-9][0-9]*`)

// matchPartialFilters matches the full module name against the partial
// filters. This ignores any version string at the end of the name and any
// missing leading parts of the module name.
//
// Note that it can potentially match multiple modules with the same name and
// different leading parts.
func (prog *prog) matchPartialFilters(moduleName string) bool {
	var lp string

	dir, lastPart := path.Split(moduleName)
	if versionRE.MatchString(lastPart) {
		if dir == "" {
			return false
		}

		dir, lastPart = path.Split(dir[:len(dir)-1])
	}

	for {
		for filt := range prog.partialFilter {
			if !prog.partialFilter[filt] {
				continue
			}

			if filt == lastPart {
				return true
			}
		}

		if dir == "" {
			return false
		}

		dir, lp = path.Split(dir[:len(dir)-1])
		lastPart = lp + "/" + lastPart
	}
}

// headerOptFuncs returns a slice of header option functions
func (prog *prog) headerOptFuncs() []col.HdrOptionFunc {
	hdrOpts := []col.HdrOptionFunc{}

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

	return hdrOpts
}

// makeReportIntroFunc returns a function that can be supplied when
// constructing a report header and will be called before the header is
// printed.
func makeReportIntroFunc(prog *prog) col.PreHdrFunc {
	const colNameIndent = 4

	maxColNameLen := 0

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

		for _, cid := range prog.columnsToShow {
			ci, err := prog.cols.GetReportableColInfo(cid)
			if err != nil {
				twc.Println(err)
				continue
			}

			twc.Println()
			twc.WrapPrefixed(fmt.Sprintf("%-*s ", maxColNameLen, cid),
				ci.FullDesc(),
				colNameIndent)
		}

		twc.Println()
	}
}

// skipModInfo returns true if the module info record should be skipped. It
// will be skippped if:
//
// The location has not been filled in (its an external module).
//
// There is a module filter and the name does not match an entry in the
// filters map.
//
// The module name is in the hideModules map
func (prog *prog) skipModInfo(mi *modInfo) bool {
	if mi.Loc == nil {
		return true
	}

	if len(prog.modFilter) > 0 && !prog.modFilter[mi.Name] {
		return true
	}

	return prog.hideModules[mi.Name]
}

// populateModInfo gathers the module info records from the modules map
// filtering them appropriately and recording them in the prog.mInfo member.
func (prog *prog) populateModInfo() {
	prog.mInfo = make([]*modInfo, 0, len(prog.mm))
	for _, mi := range prog.mm {
		if !prog.skipModInfo(mi) {
			prog.mInfo = append(prog.mInfo, mi)
		}
	}
}

// makeDotfile creates a Dotfile, a representation of the module information
// in the Graphviz DOT language which can be transformed into a picture.
func (prog *prog) makeDotfile() {
	const dotfilePattern = "gomodlayers*.gv"

	// if prog.dotFileDir is not set os.CreateTemp uses the Temp directory
	f, err := os.CreateTemp(prog.dotFileDir, dotfilePattern)
	if err != nil {
		fmt.Println("Couldn't make the Dotfile:", err)
		return
	}

	fmt.Fprintln(f, "digraph modules {")

	for _, mi := range prog.mInfo {
		name := strings.TrimPrefix(mi.Name, prog.stripPrefix)
		for _, rbMi := range mi.ReqdByDirectly {
			if prog.skipModInfo(rbMi) {
				continue
			}

			rbName := strings.TrimPrefix(rbMi.Name, prog.stripPrefix)
			fmt.Fprintf(f, "\t%q -> %q\n", rbName, name)
		}
	}

	fmt.Fprintln(f, "}")

	if err = f.Close(); err != nil {
		fmt.Println("error closing the dotfile:", err)
	}

	fmt.Println("see: ", f.Name())
}

// reportModuleInfo prints the module information
func (prog *prog) reportModuleInfo() {
	// recreate the cols with the prog value post param parsing
	prog.cols = prog.populateCols()

	reporter, err := prog.cols.MakeReport(prog,
		os.Stdout,
		prog.columnsToShow,
		prog.headerOptFuncs()...)
	if err != nil {
		fmt.Println("Couldn't make the report:", err)
		return
	}

	if err = reporter.Print(prog.mInfo, prog.makeSortCols()); err != nil {
		fmt.Println("Couldn't print the report:", err)

		return
	}
}
