package main

import (
	"strings"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/colfmt"
	"github.com/nickwells/english.mod/english"
)

// colName represents the name of a column. It is used to specify sort order
// and columns to display
type colName string

// these constants name the available columns
const (
	ColLevel        = colName("level")
	ColName         = colName("name")
	ColUseCount     = colName("use-count")
	ColUsedBy       = colName("used-by")
	ColUsesCountInt = colName("uses-count-int")
	ColUsesCountExt = colName("uses-count-ext")
	ColPackages     = colName("packages")
	ColPkgLines     = colName("lines-of-code")
)

var columnHeadings = map[colName][]string{
	ColLevel:        {"Level"},
	ColName:         {"Module name"},
	ColUseCount:     {"Count", "Used By"},
	ColUsedBy:       {"Used By"},
	ColUsesCountInt: {"Count", "Uses (int)"},
	ColUsesCountExt: {"Count", "Uses (ext)"},
	ColPackages:     {"Package", "Count"},
	ColPkgLines:     {"Package", "LoC"},
}

var columnDescription = map[colName]string{
	ColLevel: "this shows how the module relates to other" +
		" modules. Any module at level N only uses modules at level N-1" +
		" and below. The lower the level number the greater the impact of" +
		" any changes to this module will have on the whole collection of" +
		" modules." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColLevel], "/", "/"),
	ColName: "this is the module name. It includes the module" +
		" version number (if any)." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColName], "/", "/"),
	ColUseCount: "this shows how many other modules in the" +
		" collection use this module. The larger this number" +
		" the greater the impact of a change to this module." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColUseCount], "/", "/"),
	ColUsedBy: "this lists the names of the modules using this" +
		" module. Each of these will need to be changed to reflect" +
		" a change in the semantic version number of this module." +
		" These changes in turn will require a change to their semantic" +
		" version numbers and so on." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColUsedBy], "/", "/"),
	ColUsesCountInt: "this gives the number of other modules in this" +
		" collection that this module uses." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColUsesCountInt], "/", "/"),
	ColUsesCountExt: "this gives the number of modules not in this" +
		" collection that this module uses." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColUsesCountExt], "/", "/"),
	ColPackages: "this gives the number of packages that are in this" +
		" module. It will include commands (with package name 'main')." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColPackages], "/", "/"),
	ColPkgLines: "this gives the total number of lines of non-test code" +
		" in the packages." +
		"\n\n" +
		"This column is headed: " +
		english.JoinQuoted(columnHeadings[ColPkgLines], "/", "/"),
}

var reportColumns = map[colName]func(prog *prog, modules modMap) *col.Col{
	ColLevel: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{
				W: prog.reportDigits,
				DupHdlr: colfmt.DupHdlr{
					SkipDups: prog.hideDupLevels,
				},
			},
			columnHeadings[ColLevel]...)
	},
	ColName: func(_ *prog, modules modMap) *col.Col {
		return col.New(
			&colfmt.String{W: modules.findMaxNameLen()},
			columnHeadings[ColName]...)
	},
	ColUsedBy: func(_ *prog, modules modMap) *col.Col {
		return col.New(
			&colfmt.WrappedString{W: modules.findMaxNameLen()},
			columnHeadings[ColUsedBy]...)
	},
	ColUseCount: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{W: prog.reportDigits},
			columnHeadings[ColUseCount]...)
	},
	ColUsesCountInt: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{W: prog.reportDigits},
			columnHeadings[ColUsesCountInt]...)
	},
	ColUsesCountExt: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{W: prog.reportDigits},
			columnHeadings[ColUsesCountExt]...)
	},
	ColPackages: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{W: prog.reportDigits},
			columnHeadings[ColPackages]...)
	},
	ColPkgLines: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{W: prog.reportDigits},
			columnHeadings[ColPkgLines]...)
	},
}

var columnVals = map[colName]func(prog *prog, mi *modInfo) any{
	ColLevel: func(_ *prog, mi *modInfo) any { return mi.Level },
	ColName:  func(_ *prog, mi *modInfo) any { return mi.Name },
	ColUsedBy: func(_ *prog, mi *modInfo) any {
		reqdBy := make([]string, 0, len(mi.ReqdBy))
		for _, p := range mi.ReqdBy {
			reqdBy = append(reqdBy, p.Name)
		}

		return strings.Join(reqdBy, "\n")
	},
	ColUseCount: func(_ *prog, mi *modInfo) any { return len(mi.ReqdBy) },
	ColUsesCountInt: func(_ *prog, mi *modInfo) any {
		return mi.ReqCountInternal
	},
	ColUsesCountExt: func(_ *prog, mi *modInfo) any {
		return mi.ReqCountExternal
	},
	ColPackages: func(_ *prog, mi *modInfo) any { return len(mi.Packages) },
	ColPkgLines: func(_ *prog, mi *modInfo) any {
		linesOfCode := 0

		for _, pkg := range mi.Packages {
			for _, gi := range pkg.Files {
				linesOfCode += gi.LineCount
			}
		}

		return linesOfCode
	},
}

var columnCmpFunc = map[colName]func(a, b *modInfo) int{
	ColLevel: func(a, b *modInfo) int {
		return a.Level - b.Level
	},
	ColName: func(a, b *modInfo) int {
		return strings.Compare(a.Name, b.Name)
	},
	ColUseCount: func(a, b *modInfo) int {
		return len(a.ReqdBy) - len(b.ReqdBy)
	},
	ColUsedBy: func(_, _ *modInfo) int { return 0 }, // should never be used
	ColUsesCountInt: func(a, b *modInfo) int {
		return a.ReqCountInternal - b.ReqCountInternal
	},
	ColUsesCountExt: func(a, b *modInfo) int {
		return a.ReqCountExternal - b.ReqCountExternal
	},
	ColPackages: func(a, b *modInfo) int {
		return len(a.Packages) - len(b.Packages)
	},
	ColPkgLines: func(a, b *modInfo) int {
		aLen := 0
		for _, p := range a.Packages {
			aLen += p.FilesLoC
		}

		bLen := 0
		for _, p := range b.Packages {
			bLen += p.FilesLoC
		}

		return aLen - bLen
	},
}
