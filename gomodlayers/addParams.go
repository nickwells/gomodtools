package main

import (
	"slices"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v7/paction"
	"github.com/nickwells/param.mod/v7/param"
	"github.com/nickwells/param.mod/v7/psetter"
)

const (
	paramHideHeader    = "hide-header"
	paramHideIntro     = "hide-intro"
	paramHideDupLevels = "hide-dup-levels"
	paramBrief         = "brief"
	paramHeaderRepeat  = "header-repeat"
	paramSortOrder     = "sort-order"
	paramShowCols      = "show-cols"
	paramNamesByLevel  = "names-by-level"
	paramNamesOnly     = "names-only"
	paramFilter        = "filter"
	paramPartialFilter = "partial-filter"
)

// addParams will add parameters to the passed param.PSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNamesOnly,
			psetter.Nil{},
			"set the list of columns to only show the module names",
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.columnsToShow = []colName{ColName}

					return nil
				}),
		)

		ps.Add(paramHideHeader,
			psetter.Bool{Value: &prog.showHeader, Invert: true},
			"suppress the printing of the header",
			param.AltNames("hide-hdr", "no-hdr"),
		)

		ps.Add(paramHeaderRepeat,
			psetter.Int[int]{
				Value: &prog.headerRepeat,
				Checks: []check.ValCk[int]{
					check.ValGE(1),
				},
			},
			"after how many lines should the header be reprinted",
		)

		ps.Add(paramHideIntro,
			psetter.Bool{Value: &prog.showIntro, Invert: true},
			"suppress the printing of the introductory text"+
				" explaining the meaning of the report",
			param.AltNames("no-intro", "quiet"),
			param.SeeAlso(paramHideHeader, paramBrief),
		)

		ps.Add(paramBrief,
			psetter.Nil{},
			"suppress the printing of both the introductory text"+
				" and the headers",
			param.PostAction(paction.SetVal(&prog.showHeader, false)),
			param.PostAction(paction.SetVal(&prog.showIntro, false)),
		)

		ps.Add(paramHideDupLevels, psetter.Bool{Value: &prog.hideDupLevels},
			"suppress the printing of levels where the level value"+
				" is the same as on the previous line",
		)

		ps.Add(paramSortOrder,
			psetter.EnumList[colName]{
				Value: &prog.sortBy,
				AllowedVals: psetter.AllowedVals[colName]{
					ColLevel:    "in level order (lowest first)",
					ColName:     "in name order",
					ColUseCount: "in order of how heavily used the module is",
					ColUsesCountInt: "in order of how much use the module" +
						" makes of other modules in the collection",
					ColUsesCountExt: "in order of how much use the module" +
						" makes of (non-stdlib) modules not in the collection",
					ColPackages: "in order of how many packages" +
						" the module has",
					ColPkgLines: "in order of how many" +
						" lines of (non-test) code" +
						" there are in the module's packages",
				},
				Aliases: psetter.Aliases[colName]{
					"lines": {ColPkgLines},
					"loc":   {ColPkgLines},
				},
			},
			"what order should the modules be sorted when reporting",
			param.AltNames("sort-by"),
		)

		ps.Add(paramShowCols,
			psetter.EnumList[colName]{
				Value: &prog.columnsToShow,
				AllowedVals: psetter.AllowedVals[colName]{
					ColLevel: "where the module lies in the dependency" +
						" order",
					ColName:     "the module name",
					ColUseCount: "how heavily used the module is",
					ColUsedBy:   "the modules that use this",
					ColUsesCountInt: "how much use the module makes" +
						" of other modules in the collection",
					ColUsesCountExt: "how much use the module makes" +
						" of (non-stdlib) modules not in the collection",
					ColPackages: "how many packages does this module provide",
					ColPkgLines: "how many lines of" +
						" (non-test) code" +
						" there are in the module's packages",
				},
				Aliases: psetter.Aliases[colName]{
					"all": {
						ColLevel,
						ColName,
						ColUsesCountExt,
						ColUsesCountInt,
						ColUseCount,
						ColUsedBy,
						ColPackages,
						ColPkgLines,
					},
					"lines":      {ColPkgLines},
					"loc":        {ColPkgLines},
					"uses-count": {ColUsesCountExt, ColUsesCountInt},
				},
			},
			"what columns should be shown."+
				" Note that the name is always shown,"+
				" it will be added as the first column"+
				" if it is not already present",
			param.AltNames("show", "cols", "col"),
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					if slices.Contains(prog.columnsToShow, ColName) {
						return nil
					}

					prog.columnsToShow = append([]colName{ColName},
						prog.columnsToShow...)

					return nil
				}),
		)

		ps.Add(paramNamesByLevel, psetter.Nil{},
			"just show the module names in level order",
			param.PostAction(paction.SetVal(&prog.showHeader, false)),
			param.PostAction(paction.SetVal(&prog.showIntro, false)),
			param.PostAction(paction.SetVal(&prog.sortBy,
				[]colName{ColLevel, ColName})),
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.columnsToShow = []colName{ColName}

					return nil
				}),
		)

		ps.Add(paramFilter,
			psetter.Map[string]{Value: &prog.modFilter},
			"the module names to filter by."+
				" The report will only show these modules"+
				" and any modules that uses them."+
				" The notion of 'used' is recursive so that"+
				" if the filter is on module A"+
				" and module B uses A and C uses B but not A (directly)"+
				" then modules A, B and C will be shown.",
			param.AltNames("filt", "f"),
			param.SeeAlso(paramPartialFilter),
		)

		ps.Add(paramPartialFilter,
			psetter.Map[string]{Value: &prog.partialFilter},
			"the module names to filter by."+
				" This behaves like the "+paramFilter+
				" but the match is only on the end of the module name"+
				" and any version number part is excluded."+
				" so for instance a module called 'A/B/C/v2' would"+
				" be matched"+
				" by a partial filter of:\n"+
				"'A/B/C'\n"+
				"'B/C'\n"+
				"or just 'C'."+
				"\n\n"+
				"Note that a partial filter might match multiple modules"+
				" if they have differing prefixes before the start of"+
				" the partial filter.",
			param.AltNames("pf"),
			param.SeeAlso(paramFilter),
		)

		return nil
	}
}
