package main

import (
	"errors"
	"strings"

	"github.com/nickwells/col.mod/v6/col"
	"github.com/nickwells/col.mod/v6/colfmt"
	"github.com/nickwells/col.mod/v6/rptmaker"
)

// these constants name the available columns
const (
	ColLevel        = rptmaker.ColID("level")
	ColName         = rptmaker.ColID("name")
	ColUseCount     = rptmaker.ColID("use-count")
	ColUsedBy       = rptmaker.ColID("used-by")
	ColUsesCountInt = rptmaker.ColID("uses-count-int")
	ColUsesCountExt = rptmaker.ColID("uses-count-ext")
	ColPackages     = rptmaker.ColID("packages")
	ColPkgLines     = rptmaker.ColID("lines-of-code")
)

// populateCols populates and returns the report columns
func populateCols() *rptmaker.Cols[*prog, *modInfo] {
	allErrs := []error{}
	cols := rptmaker.NewCols[*prog, *modInfo]()

	allErrs = append(allErrs, cols.Add(ColLevel,
		rptmaker.NewColInfo("this shows how the module relates to other"+
			" modules. Any module at level N only uses modules at level N-1"+
			" and below. It is only used by modules at level N+1 and above."+
			" The lower the level number the greater the impact of"+
			" any changes to this module will have on the whole collection of"+
			" modules.",
			[]string{"Level"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(
					&colfmt.Int{
						W: prog.reportDigits,
						DupHdlr: colfmt.DupHdlr{
							SkipDups: prog.hideDupLevels,
						},
					},
					headings...)
			},
			func(mi *modInfo) any { return mi.Level },
			func(a, b *modInfo) int {
				return a.Level - b.Level
			})))

	allErrs = append(allErrs, cols.Add(ColName,
		rptmaker.NewColInfo("this is the module name. It includes the module"+
			" version number (if any).",
			[]string{"Module name"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.String{W: prog.maxNameLen}, headings...)
			},
			func(mi *modInfo) any { return mi.Name },
			func(a, b *modInfo) int { return strings.Compare(a.Name, b.Name) })))

	allErrs = append(allErrs, cols.Add(ColUseCount,
		rptmaker.NewColInfo("this shows how many other modules in the"+
			" collection use this module. The larger this number"+
			" the greater the impact of a change to this module.",
			[]string{"Count", "Used By"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			func(mi *modInfo) any { return len(mi.ReqdBy) },
			func(a, b *modInfo) int { return len(a.ReqdBy) - len(b.ReqdBy) })))

	allErrs = append(allErrs, cols.Add(ColUsedBy,
		rptmaker.NewColInfo(
			"this lists the names of the modules using this"+
				" module both directly and indirectly (through the use"+
				" of a package that itself uses this package)."+
				" Each of these will need to be changed to reflect"+
				" any change in the semantic version number of this"+
				" module. These changes in turn will require a change to"+
				" their semantic version numbers and so on.",
			[]string{"Used By"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.WrappedString{W: prog.maxNameLen},
					headings...)
			},
			func(mi *modInfo) any {
				reqdBy := make([]string, 0, len(mi.ReqdBy))
				for _, p := range mi.ReqdBy {
					reqdBy = append(reqdBy, p.Name)
				}

				return strings.Join(reqdBy, "\n")
			},
			nil)))

	allErrs = append(allErrs, cols.Add(ColUsesCountInt,
		rptmaker.NewColInfo(
			"this gives the number of other modules in this"+
				" collection that this module uses.",
			[]string{"Count", "Uses (int)"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			func(mi *modInfo) any { return mi.ReqCountInt },
			func(a, b *modInfo) int { return a.ReqCountInt - b.ReqCountInt },
		)))

	allErrs = append(allErrs, cols.Add(ColUsesCountExt,
		rptmaker.NewColInfo(
			"this gives the number of modules not in this"+
				" collection that this module uses.",
			[]string{"Count", "Uses (ext)"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			func(mi *modInfo) any { return mi.ReqCountExt },
			func(a, b *modInfo) int { return a.ReqCountExt - b.ReqCountExt },
		)))

	allErrs = append(allErrs, cols.Add(ColPackages,
		rptmaker.NewColInfo(
			"this gives the number of packages that are in this"+
				" module. It will include commands (with package name 'main').",
			[]string{"Package", "Count"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			func(mi *modInfo) any { return len(mi.Packages) },
			func(a, b *modInfo) int { return len(a.Packages) - len(b.Packages) },
		)))

	allErrs = append(allErrs, cols.Add(ColPkgLines,
		rptmaker.NewColInfo(
			"this gives the total number of lines of non-test code"+
				" in the packages.",
			[]string{"Package", "LoC"},
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			func(mi *modInfo) any {
				return mi.LinesOfCode
			},
			func(a, b *modInfo) int {
				return a.LinesOfCode - b.LinesOfCode
			},
		)))

	allErrs = append(allErrs,
		cols.AddAlias(rptmaker.ColID("lines"), ColPkgLines))

	allErrs = append(allErrs,
		cols.AddAlias(rptmaker.ColID("loc"), ColPkgLines))

	allErrs = append(allErrs,
		cols.AddReportableAlias(rptmaker.ColID("all"),
			ColLevel,
			ColName,
			ColUsesCountExt,
			ColUsesCountInt,
			ColUseCount,
			ColUsedBy,
			ColPackages,
			ColPkgLines,
		))

	if errs := errors.Join(allErrs...); errs != nil {
		panic(errs)
	}

	return cols
}
