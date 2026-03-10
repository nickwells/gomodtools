package main

import (
	"strings"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/colfmt"
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

var reportColumns = map[colName]func(prog *prog, modules modMap) *col.Col{
	ColLevel: func(prog *prog, _ modMap) *col.Col {
		return col.New(
			&colfmt.Int{
				W: prog.reportDigits,
				DupHdlr: colfmt.DupHdlr{
					SkipDups: prog.hideDupLevels,
				},
			},
			"Level")
	},
	ColName: func(_ *prog, modules modMap) *col.Col {
		return col.New(
			&colfmt.String{W: modules.findMaxNameLen()},
			"Module name")
	},
	ColUsedBy: func(_ *prog, modules modMap) *col.Col {
		return col.New(
			&colfmt.WrappedString{W: modules.findMaxNameLen()},
			"Used By")
	},
	ColUseCount: func(prog *prog, _ modMap) *col.Col {
		return col.New(&colfmt.Int{W: prog.reportDigits}, "Count", "Used By")
	},
	ColUsesCountInt: func(prog *prog, _ modMap) *col.Col {
		return col.New(&colfmt.Int{W: prog.reportDigits}, "Count", "Uses (int)")
	},
	ColUsesCountExt: func(prog *prog, _ modMap) *col.Col {
		return col.New(&colfmt.Int{W: prog.reportDigits}, "Count", "Uses (ext)")
	},
	ColPackages: func(prog *prog, _ modMap) *col.Col {
		return col.New(&colfmt.Int{W: prog.reportDigits}, "Package", "Count")
	},
	ColPkgLines: func(prog *prog, _ modMap) *col.Col {
		return col.New(&colfmt.Int{W: prog.reportDigits}, "Package", "LoC")
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
