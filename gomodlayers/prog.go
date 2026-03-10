package main

import (
	"path"
	"regexp"
)

// prog holds program parameters and status
type prog struct {
	hideDupLevels bool
	showIntro     bool
	showHeader    bool

	sortBy []colName

	modFilter     map[string]bool
	partialFilter map[string]bool
	columnsToShow []colName

	reportDigits int
	headerRepeat int
}

// newProg returns a new Prog instance with the default values set
func newProg() *prog {
	const dfltDigitsToShow = 5

	return &prog{
		showIntro:  true,
		showHeader: true,

		sortBy:        []colName{ColLevel},
		columnsToShow: []colName{ColLevel, ColName, ColUseCount},

		modFilter:    map[string]bool{},
		reportDigits: dfltDigitsToShow,
	}
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
