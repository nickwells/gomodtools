package main

// gomodlayers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// Created: Thu Mar 28 12:13:29 2019

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

	modules := parseAllGoModFiles(ps.Remainder())
	modules.calcLevels()
	modules.calcReqCount()
	modules.expandModFilters()
	modules.reportModuleInfo()
}

// parseAllGoModFiles will process the list of filenames, opening each one in
// turn and populating the moduleInfo map. If any filename doesn't end with
// go.mod then that is added to the end of the path before further processing
func parseAllGoModFiles(goModFilenames []string) ModMap {
	modules := ModMap{}
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

		mi := parseGoModFile(modules, f, location.New(fname))
		if mi == nil {
			fmt.Fprintf(os.Stderr, "Error: No module defined in: %q\n", fname)
			continue
		}
	}
	return modules
}

// calcLevels will repeatedly go over the modules resetting the level to be
// one greater than that of the highest level module which it requires. It
// keeps on doing this until it has made no further changes; this should be
// sufficient as Go does not permit loops in module requirements but to cope
// with bugs in module specs we abort if the max level observed is greater
// than the total number of modules being considered.
func (modules ModMap) calcLevels() {
	levelChange := true
	maxLevel := 0
	for levelChange && maxLevel <= len(modules) {
		levelChange = false
		for _, mi := range modules {
			if mi.calcLevel() {
				levelChange = true
				if mi.Level > maxLevel {
					maxLevel = mi.Level
				}
			}
		}
	}
}

// calcReqCount will calculate the number of internal and external
// requirements for each module. If a required module has no location set
// then it is taken to be an external requireement.
func (modules ModMap) calcReqCount() {
	for _, mi := range modules {
		mi.setReqCounts()
	}
}

// findMaxNameLen returns the length of the longest module name
func (modules ModMap) findMaxNameLen() int {
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
func (modules ModMap) makeModInfoSlice(order string) []*ModInfo {
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

// expandModFilters takes the initial set of modFilters and adds all the
// other modules that it is required by.
func (modules ModMap) expandModFilters() {
	if len(modFilter) == 0 {
		return
	}

	for _, mi := range modules.makeModInfoSlice(ColLevel) {
		if modFilter[mi.Name] {
			addReqsToFilters(mi)
		}
	}
}
