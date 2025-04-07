package main

import (
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v6/paction"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	paramHideHeader    = "hide-header"
	paramHideIntro     = "hide-intro"
	paramHideDupLevels = "hide-dup-levels"
	paramBrief         = "brief"
	paramNoSkips       = "no-skips"
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
			"reset the list of columns to only show the module names",
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					prog.columnsToShow = map[string]bool{
						ColName: true,
					}

					return nil
				}),
		)

		ps.Add(paramHideHeader,
			psetter.Bool{Value: &prog.showHeader, Invert: true},
			"suppress the printing of the header",
			param.AltNames("hide-hdr", "no-hdr"),
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

		ps.Add(paramNoSkips,
			psetter.Bool{Value: &prog.canSkipCols, Invert: true},
			"don't skip the printing of columns where the row"+
				" value is the same as on the previous line."+
				" Note that this value overrides"+
				" the '"+paramHideDupLevels+"' parameter (if set)",
			param.PostAction(paction.SetVal(&prog.hideDupLevels, false)),
			param.AltNames("dont-skip-cols", "dont-skip"),
		)

		ps.Add(paramSortOrder,
			psetter.Enum[string]{
				Value: &prog.sortBy,
				AllowedVals: psetter.AllowedVals[string]{
					ColLevel:    "in level order (lowest first)",
					ColName:     "in name order",
					ColUseCount: "in order of how heavily used the module is",
					ColUsesCountInt: "in order of how much use the module" +
						" makes of other modules in the collection",
					ColUsesCountExt: "in order of how much use the module" +
						" makes of modules not in the collection",
				},
			},
			"what order should the modules be sorted when reporting",
			param.AltNames("sort-by"),
		)

		ps.Add(paramShowCols,
			psetter.EnumMap[string]{
				Value: &prog.columnsToShow,
				AllowedVals: psetter.AllowedVals[string]{
					ColLevel: "where the module lies in the dependency" +
						" order",
					ColUseCount: "how heavily used the module is",
					ColUsedBy:   "the modules that use this",
					ColUsesCountInt: "how much use the module makes" +
						" of other modules in the collection",
					ColUsesCountExt: "how much use the module makes" +
						" of modules not in the collection",
					ColPackages: "how many packages does this module provide",
					ColPkgLines: "how many lines of code the module" +
						" packages provide",
				},
				Aliases: psetter.Aliases[string]{
					"all": {
						ColLevel,
						ColUseCount,
						ColUsedBy,
						ColUsesCountExt,
						ColUsesCountInt,
						ColPackages,
						ColPkgLines,
					},
					"lines":      {ColPkgLines},
					"uses-count": {ColUsesCountExt, ColUsesCountInt},
				},
				AllowHiddenMapEntries: true,
			},
			"what columns should be shown (note that the name is always shown)",
			param.AltNames("show", "cols", "col"),
		)

		ps.Add(paramNamesByLevel, psetter.Nil{},
			"just show the module names in level order",
			param.PostAction(paction.SetVal(&prog.showHeader, false)),
			param.PostAction(paction.SetVal(&prog.showIntro, false)),
			param.PostAction(paction.SetVal(&prog.sortBy, ColLevel)),
			param.PostAction(
				func(_ location.L, _ *param.ByName, _ []string) error {
					prog.columnsToShow = map[string]bool{
						ColName: true,
					}

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

		// allow trailing arguments
		err := ps.SetNamedRemHandler(param.NullRemHandler{}, "go.mod-files")
		if err != nil {
			return err
		}

		return nil
	}
}
