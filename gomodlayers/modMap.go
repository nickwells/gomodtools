package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nickwells/errutil.mod/errutil"
	"github.com/nickwells/location.mod/location"
)

// modMap associates names with the information from go.mod files
type modMap map[string]*modInfo

// populate fills the modMap with the module information from the given
// files. Note that the 'file' names can be directory names in which case the
// name of the Go module file is added.
func (mm modMap) populate(fNames []string) *errutil.ErrMap {
	const goMod = "go.mod"

	errMap := errutil.NewErrMap()

	for _, fname := range fNames {
		if !strings.HasSuffix(fname, goMod) {
			fname = filepath.Join(fname, goMod)
		}

		contents, err := os.ReadFile(fname) //nolint:gosec
		if err != nil {
			errMap.AddError(fname, err)

			continue
		}

		mi, err := parseGoModFile(mm, contents, location.New(fname))
		if err != nil {
			errMap.AddError(fname, err)

			continue
		}

		mi.getPackageInfo(filepath.Dir(fname))
	}

	mm.sortReqdByNames()

	return errMap
}

// sortReqdByNames sorts the cross reference entries for each modInfo
// entry in the modules map. the entries are sorted by the module name.
func (mm modMap) sortReqdByNames() {
	for _, mi := range mm {
		mi.sortCrossRefs()
	}
}

// findMaxNameLen returns the length of the longest module name
func (mm modMap) findMaxNameLen() int {
	maxLen := 0
	for _, mi := range mm {
		maxLen = max(len(mi.Name), maxLen)
	}

	return maxLen
}

// calcLevels will repeatedly go over the modules resetting the level to be
// one greater than that of the highest level module which it requires. It
// keeps on doing this until it has made no further changes; this should be
// sufficient as Go does not permit loops in module requirements but to cope
// with bugs in module specs we abort if the max level observed is greater
// than the total number of modules being considered.
func (mm modMap) calcLevels() {
	levelChange := true
	maxLevel := 0

	for levelChange && maxLevel <= len(mm) {
		levelChange = false

		for _, mi := range mm {
			if mi.calcLevel() {
				levelChange = true
				maxLevel = max(mi.Level, maxLevel)
			}
		}
	}
}

// calcReqCount will calculate the number of internal and external
// requirements for each module. If a required module has no location set
// then it is taken to be an external requireement.
func (mm modMap) calcReqCount() {
	for _, mi := range mm {
		mi.setReqCounts()
	}
}
