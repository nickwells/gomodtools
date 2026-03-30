package main

import (
	"slices"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/col.mod/v6/rptmaker"
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

type sortWay = rptmaker.SortWay

// addParams will add parameters to the passed param.PSet
func addParams(prog *prog) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(paramNamesOnly,
			psetter.Nil{},
			"set the list of columns to only show the module names",
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.columnsToShow = []rptmaker.ColID{ColName}

					return nil
				}),
		)

		ps.Add(paramHideHeader,
			psetter.Bool{Value: &prog.showHeader, Invert: true},
			"suppress the printing of the header",
			param.AltNames("hide-hdr", "no-hdr"),
			param.SeeAlso(paramHideIntro, paramBrief),
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
			psetter.TaggedValueList[rptmaker.ColID, sortWay]{
				Value: &prog.sortBy,
				AllowedVals: psetter.AllowedVals[rptmaker.ColID](
					prog.cols.Sortable(),
				),
				Aliases: psetter.Aliases[rptmaker.ColID](
					prog.cols.SortableAliases(),
				),
				TagAllowedVals: psetter.AllowedVals[sortWay](
					rptmaker.AllowedSortDirections(),
				),
				TagAliases: psetter.Aliases[sortWay](
					rptmaker.SortDirectionAliases(),
				),
				TagListSeparator: psetter.StrListSeparator{Sep: "|"},
				TagChecks: []check.ValCk[[]sortWay]{
					check.SliceLength[[]sortWay](check.ValBetween(0, 1)),
				},
			},
			"what order should the modules be sorted when reporting",
			param.AltNames("sort-by", "order-by", "order"),
		)

		ps.Add(paramShowCols,
			psetter.EnumList[rptmaker.ColID]{
				Value: &prog.columnsToShow,
				AllowedVals: psetter.AllowedVals[rptmaker.ColID](
					prog.cols.Reportable(),
				),
				Aliases: psetter.Aliases[rptmaker.ColID](
					prog.cols.ReportableAliases(),
				),
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

					prog.columnsToShow = append([]rptmaker.ColID{ColName},
						prog.columnsToShow...)

					return nil
				}),
		)

		ps.Add(paramNamesByLevel, psetter.Nil{},
			"just show the module names in level order",
			param.PostAction(paction.SetVal(&prog.showHeader, false)),
			param.PostAction(paction.SetVal(&prog.showIntro, false)),
			param.PostAction(paction.SetVal(&prog.sortBy,
				[]sortCol{
					{Value: ColLevel},
					{Value: ColName},
				})),
			param.PostAction(
				func(_ location.L, _ *param.BaseParam, _ []string) error {
					prog.columnsToShow = []rptmaker.ColID{ColName}

					return nil
				}),
		)

		ps.Add(paramFilter,
			psetter.Map[string]{Value: &prog.modFilter},
			"the module names to filter by."+
				" The report will only show these modules"+
				" and any modules that use them."+
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
