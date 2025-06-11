package main

// gomodlayers

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/nickwells/location.mod/location"
)

// Created: Thu Mar 28 12:13:29 2019

func main() {
	prog := newProg()
	ps := makeParamSet(prog)

	ps.Parse()

	modules := parseAllGoModFiles(ps.Remainder())
	modules.calcLevels()
	modules.calcReqCount()
	modules.expandModFilters(prog)
	modules.reportModuleInfo(prog)
}

// parseAllGoModFiles will process the list of filenames, opening each one in
// turn and populating the moduleInfo map. If any filename doesn't end with
// go.mod then that is added to the end of the path before further processing
func parseAllGoModFiles(goModFilenames []string) modMap {
	modules := modMap{}

	const goMod = "go.mod"

	for _, fname := range goModFilenames {
		if !strings.HasSuffix(fname, goMod) {
			fname = filepath.Join(fname, goMod)
		}

		contents, err := os.ReadFile(fname) //nolint:gosec
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		mi, err := parseGoModFile(modules, contents, location.New(fname))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: No module defined in: %q: %s\n",
				fname, err)
			continue
		}

		mi.getPackageInfo(filepath.Dir(fname))
	}

	sortReqdByNames(modules)

	return modules
}

// sortReqdByNames sorts the entries in the ReqdBy slices for each modInfo
// entry in the modules map. the entries are sorted by the module name.
func sortReqdByNames(modules modMap) {
	for _, mi := range modules {
		slices.SortFunc(mi.ReqdBy,
			func(a, b *modInfo) int {
				return strings.Compare(a.Name, b.Name)
			})
	}
}

// calcLevels will repeatedly go over the modules resetting the level to be
// one greater than that of the highest level module which it requires. It
// keeps on doing this until it has made no further changes; this should be
// sufficient as Go does not permit loops in module requirements but to cope
// with bugs in module specs we abort if the max level observed is greater
// than the total number of modules being considered.
func (modules modMap) calcLevels() {
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
func (modules modMap) calcReqCount() {
	for _, mi := range modules {
		mi.setReqCounts()
	}
}

// findMaxNameLen returns the length of the longest module name
func (modules modMap) findMaxNameLen() uint {
	l := 0
	for _, mi := range modules {
		if len(mi.Name) > l {
			l = len(mi.Name)
		}
	}

	return uint(l) //nolint:gosec
}

// makeModInfoSlice returns the modules map as a slice of ModInfo
// pointers. The slice will be sorted according to the value of the sort
// parameter
func (modules modMap) makeModInfoSlice(order string) []*modInfo {
	ms := slices.Collect(maps.Values(modules))

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
	case ColPackages:
		sort.Slice(ms,
			func(i, j int) bool { return lessByPackages(ms, i, j) })
	case ColPkgLines:
		sort.Slice(ms,
			func(i, j int) bool { return lessByPkgLines(ms, i, j) })
	}

	return ms
}

// expandModFilters takes the initial set of modFilters and adds all the
// other modules that it is required by.
func (modules modMap) expandModFilters(prog *prog) {
	if len(prog.modFilter) == 0 &&
		len(prog.partialFilter) == 0 {
		return
	}

	for _, mi := range modules.makeModInfoSlice(ColLevel) {
		if prog.modFilter[mi.Name] {
			prog.addReqsToFilters(mi)
		} else {
			if prog.matchPartialFilters(mi.Name) {
				prog.modFilter[mi.Name] = true
				prog.addReqsToFilters(mi)
			}
		}
	}
}
