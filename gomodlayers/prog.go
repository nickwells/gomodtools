package main

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/rptmaker"
	"github.com/nickwells/param.mod/v7/psetter"
	"github.com/nickwells/twrap.mod/twrap"
)

// sortCol is a type alias for the TaggedEnum
type sortCol = psetter.TaggedValue[rptmaker.ColID, rptmaker.SortWay]

// prog holds program parameters and status
type prog struct {
	hideDupLevels bool
	showIntro     bool
	showHeader    bool

	sortBy []sortCol

	modFilter     map[string]bool
	partialFilter map[string]bool
	columnsToShow []rptmaker.ColID

	mInfo []*modInfo

	maxNameLen int

	reportDigits int
	headerRepeat int

	cols *rptmaker.Cols[*prog, *modInfo]
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	const dfltDigitsToShow = 5

	return &prog{
		showIntro:  true,
		showHeader: true,

		sortBy:        []sortCol{{Value: ColLevel}, {Value: ColName}},
		columnsToShow: []rptmaker.ColID{ColLevel, ColName, ColUseCountTotal},

		modFilter:     map[string]bool{},
		partialFilter: map[string]bool{},

		reportDigits: dfltDigitsToShow,

		cols: populateCols(),
	}
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

// populateModInfo gathers the module info records from the ModMap filtering
// them appropriately and recording them in the prog.mInfo member.
func (prog *prog) populateModInfo(modules modMap) {
	prog.mInfo = make([]*modInfo, 0, len(modules))
	for _, mi := range slices.Collect(maps.Values(modules)) {
		if !prog.skipModInfo(mi) && !prog.matchPartialFilters(mi.Name) {
			prog.mInfo = append(prog.mInfo, mi)
		}
	}
}
