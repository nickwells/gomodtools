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
	ColLevel          = rptmaker.ColID("level")
	ColName           = rptmaker.ColID("name")
	ColUseCountDirect = rptmaker.ColID("direct-use-count")
	ColUseCountTotal  = rptmaker.ColID("use-count")
	ColUsedBy         = rptmaker.ColID("used-by")
	ColUsedByDirectly = rptmaker.ColID("used-by-directly")
	ColUsesCountInt   = rptmaker.ColID("uses-count-int")
	ColUsesCountExt   = rptmaker.ColID("uses-count-ext")
	ColUsesDirectly   = rptmaker.ColID("uses-directly")
	ColUses           = rptmaker.ColID("uses")
	ColPackages       = rptmaker.ColID("packages")
	ColPkgLines       = rptmaker.ColID("lines-of-code")

	AliasLines  = rptmaker.ColID("lines")
	AliasLoC    = rptmaker.ColID("loc")
	AliasFull   = rptmaker.ColID("full")
	AliasDirect = rptmaker.ColID("direct")

	indirectSeparator = "** Indirect **"
	externalSeparator = "** External **"

	separatorCount = 2 // the blank line plus the separator itself
)

// addColLevel adds the level column to the supplied cols parameter.
func addColLevel(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColLevel,
		rptmaker.NewColInfo("this shows how the module relates to other"+
			" modules. Any module at level N only uses modules at level N-1"+
			" and below. It is only used by modules at level N+1 and above."+
			" The lower the level number the greater the impact of"+
			" any changes to this module will have on the whole collection of"+
			" modules.",
			[]string{"Level"},
			// mkCol
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
			// colVal
			func(mi *modInfo) any { return mi.Level },
			// cmpVals
			func(a, b *modInfo) int {
				return a.Level - b.Level
			}))
}

// addColName adds the name column to the supplied cols parameter.
func addColName(p *prog, cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColName,
		rptmaker.NewColInfo("this is the module name. It includes the module"+
			" version number (if any).",
			[]string{"Module name"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.String{W: prog.maxNameLen}, headings...)
			},
			// colVal
			func(mi *modInfo) any { return strings.TrimPrefix(mi.Name, p.stripPrefix) },
			// cmpVals
			func(a, b *modInfo) int {
				return strings.Compare(a.Name, b.Name)
			}))
}

// addColUseCountDirect adds the useCountDirect column to the supplied cols
// parameter.
func addColUseCountDirect(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUseCountDirect,
		rptmaker.NewColInfo("this shows how many other modules in the"+
			" collection use this module. The larger this number"+
			" the greater the impact of a change to this module.",
			[]string{"Count", "Used By", "Directly"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any { return len(mi.ReqdByDirectly) },
			// cmpVals
			func(a, b *modInfo) int {
				return len(a.ReqdByDirectly) - len(b.ReqdByDirectly)
			}))
}

// addColUseCountTotal adds the useCountTotal column to the supplied cols
// parameter.
func addColUseCountTotal(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUseCountTotal,
		rptmaker.NewColInfo("this shows how many other modules in the"+
			" collection use this module, either directly or indirectly."+
			" The larger this number the greater the impact of a change"+
			" to this module.",
			[]string{"Count", "Used By", "Total"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any {
				return len(mi.ReqdByDirectly) + len(mi.ReqdByIndirectly)
			},
			// cmpVals
			func(a, b *modInfo) int {
				aTotUseCount := len(a.ReqdByDirectly) + len(a.ReqdByIndirectly)
				bTotUseCount := len(b.ReqdByDirectly) + len(b.ReqdByIndirectly)

				return aTotUseCount - bTotUseCount
			}))
}

// addColUsedBy adds the usedBy column to the supplied cols parameter.
func addColUsedBy(p *prog, cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUsedBy,
		rptmaker.NewColInfo(
			"this lists the names of the modules using this"+
				" module both directly and indirectly (through the use"+
				" of a package that itself uses this package)."+
				" Each of these will need to be changed to reflect"+
				" any change in the semantic version number of this"+
				" module. These changes in turn will require a change to"+
				" their semantic version numbers and so on.",
			[]string{"Used By"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.WrappedString{W: prog.maxNameLen},
					headings...)
			},
			// colVal
			func(mi *modInfo) any {
				reqdBy := make([]string, 0,
					len(mi.ReqdByDirectly)+
						len(mi.ReqdByIndirectly)+
						separatorCount)
				for _, rb := range mi.ReqdByDirectly {
					reqdBy = append(reqdBy,
						strings.TrimPrefix(rb.Name, p.stripPrefix))
				}

				if len(mi.ReqdByIndirectly) > 0 {
					if len(reqdBy) > 0 {
						reqdBy = append(reqdBy, "")
					}

					reqdBy = append(reqdBy, indirectSeparator)
					for _, rb := range mi.ReqdByIndirectly {
						reqdBy = append(reqdBy,
							strings.TrimPrefix(rb.Name, p.stripPrefix))
					}
				}

				return strings.Join(reqdBy, "\n")
			},
			nil))
}

// addColUsedByDirectly adds the usedByDirectly column to the supplied cols
// parameter.
func addColUsedByDirectly(p *prog, cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUsedByDirectly,
		rptmaker.NewColInfo(
			"this lists the names of the modules using this"+
				" module directly. Each of these may need to"+
				" be changed to reflect any change in the API"+
				" or behaviour of this module or to use any"+
				" new features.",
			[]string{"Used By", "Directly"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.WrappedString{W: prog.maxNameLen},
					headings...)
			},
			// colVal
			func(mi *modInfo) any {
				reqdBy := make([]string, 0, len(mi.ReqdByDirectly))
				for _, rb := range mi.ReqdByDirectly {
					reqdBy = append(reqdBy,
						strings.TrimPrefix(rb.Name, p.stripPrefix))
				}

				return strings.Join(reqdBy, "\n")
			},
			nil))
}

// addColUsesCountInt adds the usesCountInt column to the supplied cols
// parameter.
func addColUsesCountInt(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUsesCountInt,
		rptmaker.NewColInfo(
			"this gives the number of other modules in this"+
				" collection that this module uses directly.",
			[]string{"Count", "Uses", "(int)"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any { return mi.ReqCountInt },
			// cmpVals
			func(a, b *modInfo) int { return a.ReqCountInt - b.ReqCountInt },
		))
}

// addColUsesCountExt adds the usesCountExt column to the supplied cols
// parameter.
func addColUsesCountExt(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUsesCountExt,
		rptmaker.NewColInfo(
			"this gives the number of modules not in this"+
				" collection that this module uses directly.",
			[]string{"Count", "Uses", "(ext)"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any { return mi.ReqCountExt },
			// cmpVals
			func(a, b *modInfo) int { return a.ReqCountExt - b.ReqCountExt },
		))
}

// addColUsesDirectly adds the usesDirectly column to the supplied cols
// parameter.
func addColUsesDirectly(p *prog, cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUsesDirectly,
		rptmaker.NewColInfo(
			"this lists the names of the modules that"+
				" this module uses directly.",
			[]string{"Uses", "Directly"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.WrappedString{W: prog.maxNameLen},
					headings...)
			},
			// colVal
			func(mi *modInfo) any {
				uses := make([]string, 0,
					len(mi.DirectReqs)+
						separatorCount)
				usesExternal := []string{}

				for _, r := range mi.DirectReqs {
					if r.Loc == nil {
						usesExternal = append(usesExternal,
							strings.TrimPrefix(r.Name, p.stripPrefix))

						continue
					}

					uses = append(uses,
						strings.TrimPrefix(r.Name, p.stripPrefix))
				}

				if len(usesExternal) > 0 {
					if len(uses) > 0 {
						uses = append(uses, "")
					}

					uses = append(uses, externalSeparator)
					uses = append(uses, usesExternal...)
				}

				return strings.Join(uses, "\n")
			},
			nil))
}

// addColUses adds the uses column to the supplied cols parameter.
func addColUses(p *prog, cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColUses,
		rptmaker.NewColInfo(
			"this lists the names of the modules that"+
				" this module uses both directly and indirectly.",
			[]string{"Uses"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.WrappedString{W: prog.maxNameLen},
					headings...)
			},
			// colVal
			func(mi *modInfo) any {
				uses := make([]string, 0,
					len(mi.DirectReqs)+
						len(mi.IndirectReqs)+
						separatorCount+ // the external separator
						separatorCount) // the indirect separator

				usesExternal := []string{}

				for _, r := range mi.DirectReqs {
					if r.Loc == nil {
						usesExternal = append(usesExternal,
							strings.TrimPrefix(r.Name, p.stripPrefix))

						continue
					}

					uses = append(uses,
						strings.TrimPrefix(r.Name, p.stripPrefix))
				}

				if len(mi.IndirectReqs) > 0 {
					usesIndirect := make([]string, 0, len(mi.IndirectReqs))
					for _, r := range mi.IndirectReqs {
						if r.Loc == nil {
							usesExternal = append(usesExternal,
								strings.TrimPrefix(r.Name, p.stripPrefix))

							continue
						}

						usesIndirect = append(usesIndirect,
							strings.TrimPrefix(r.Name, p.stripPrefix))
					}

					if len(usesIndirect) > 0 {
						if len(uses) > 0 {
							uses = append(uses, "")
						}

						uses = append(uses, indirectSeparator)
						uses = append(uses, usesIndirect...)
					}
				}

				if len(usesExternal) > 0 {
					if len(uses) > 0 {
						uses = append(uses, "")
					}

					uses = append(uses, externalSeparator)
					uses = append(uses, usesExternal...)
				}

				return strings.Join(uses, "\n")
			},
			nil))
}

// addColPackages adds the packages column to the supplied cols parameter.
func addColPackages(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColPackages,
		rptmaker.NewColInfo(
			"this gives the number of packages that are in this"+
				" module. It will include commands (with package name 'main').",
			[]string{"Package", "Count"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any { return len(mi.Packages) },
			// cmpVals
			func(a, b *modInfo) int {
				return len(a.Packages) - len(b.Packages)
			},
		))
}

// addColPkgLines adds the pkgLines column to the supplied cols parameter.
func addColPkgLines(cols *rptmaker.Cols[*prog, *modInfo]) error {
	return cols.Add(ColPkgLines,
		rptmaker.NewColInfo(
			"this gives the total number of lines of non-test code"+
				" in the packages.",
			[]string{"Package", "LoC"},
			// mkCol
			func(prog *prog, headings []string) *col.Col {
				return col.New(&colfmt.Int{W: prog.reportDigits}, headings...)
			},
			// colVal
			func(mi *modInfo) any {
				return mi.LinesOfCode
			},
			// cmpVals
			func(a, b *modInfo) int {
				return a.LinesOfCode - b.LinesOfCode
			},
		))
}

// populateCols populates and returns the report columns
func (p *prog) populateCols() *rptmaker.Cols[*prog, *modInfo] {
	allErrs := []error{}
	cols := rptmaker.NewCols[*prog, *modInfo]()

	allErrs = append(allErrs, addColLevel(cols))
	allErrs = append(allErrs, addColName(p, cols))
	allErrs = append(allErrs, addColUseCountDirect(cols))
	allErrs = append(allErrs, addColUseCountTotal(cols))
	allErrs = append(allErrs, addColUsedBy(p, cols))
	allErrs = append(allErrs, addColUsedByDirectly(p, cols))
	allErrs = append(allErrs, addColUsesCountInt(cols))
	allErrs = append(allErrs, addColUsesCountExt(cols))
	allErrs = append(allErrs, addColUsesDirectly(p, cols))
	allErrs = append(allErrs, addColUses(p, cols))
	allErrs = append(allErrs, addColPackages(cols))
	allErrs = append(allErrs, addColPkgLines(cols))

	allErrs = append(allErrs, cols.AddAlias(AliasLines, ColPkgLines))
	allErrs = append(allErrs, cols.AddAlias(AliasLoC, ColPkgLines))

	allErrs = append(allErrs, cols.AddReportableAlias(AliasFull,
		ColLevel,
		ColName,
		ColUsesCountExt,
		ColUsesCountInt,
		ColUses,
		ColUseCountTotal,
		ColUseCountDirect,
		ColUsedBy,
		ColPackages,
		ColPkgLines,
	))

	allErrs = append(allErrs, cols.AddReportableAlias(AliasDirect,
		ColLevel,
		ColName,
		ColUsesCountExt,
		ColUsesCountInt,
		ColUsesDirectly,
		ColUseCountDirect,
		ColUsedByDirectly,
		ColPackages,
		ColPkgLines,
	))

	if errs := errors.Join(allErrs...); errs != nil {
		panic(errs)
	}

	return cols
}
