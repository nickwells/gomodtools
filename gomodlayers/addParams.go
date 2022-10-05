package main

import (
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
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
)

// addParams will add parameters to the passed param.PSet
func addParams(ps *param.PSet) error {
	ps.Add(paramNamesOnly, psetter.Nil{},
		"reset the list of columns to only show the module names",
		param.PostAction(func(_ location.L, _ *param.ByName, _ []string) error {
			columnsToShow = map[string]bool{
				ColName: true,
			}
			return nil
		}),
	)

	ps.Add(paramHideHeader, psetter.Bool{Value: &showHeader, Invert: true},
		"suppress the printing of the header",
		param.AltNames("hide-hdr", "no-hdr"),
	)
	ps.Add(paramHideIntro, psetter.Bool{Value: &showIntro, Invert: true},
		"suppress the printing of the introductory text"+
			" explaining the meaning of the report",
		param.AltNames("no-intro"),
	)
	ps.Add(paramBrief, psetter.Nil{},
		"suppress the printing of both the introductory text and the headers",
		param.PostAction(paction.SetBool(&showHeader, false)),
		param.PostAction(paction.SetBool(&showIntro, false)),
	)

	ps.Add(paramHideDupLevels, psetter.Bool{Value: &hideDupLevels},
		"suppress the printing of levels where the level value"+
			" is the same as on the previous line",
	)

	ps.Add(paramNoSkips, psetter.Bool{Value: &canSkipCols, Invert: true},
		"don't skip the printing of columns where the row"+
			" value is the same as on the previous line."+
			" Note that this value overrides"+
			" the '"+paramHideDupLevels+"' parameter (if set)",
		param.PostAction(paction.SetBool(&hideDupLevels, false)),
		param.AltNames("dont-skip-cols", "dont-skip"),
	)

	ps.Add(paramSortOrder,
		psetter.Enum{
			Value: &sortBy,
			AllowedVals: psetter.AllowedVals{
				ColLevel:    "in level order (lowest first)",
				ColName:     "in name order",
				ColUseCount: "in order of how heavily used the module is",
				ColUsesCountInt: "in order of how much use the module makes" +
					" of other modules in the collection",
				ColUsesCountExt: "in order of how much use the module makes" +
					" of modules not in the collection",
			},
		},
		"what order should the modules be sorted when reporting",
		param.AltNames("sort-by"),
	)

	ps.Add(paramShowCols,
		psetter.EnumMap{
			Value: &columnsToShow,
			AllowedVals: psetter.AllowedVals{
				ColLevel:    "where the module lies in the dependency order",
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
			Aliases: psetter.Aliases{
				"all": {
					ColLevel,
					ColUseCount,
					ColUsedBy,
					ColUsesCountExt,
					ColUsesCountInt,
					ColPackages,
					ColPkgLines,
				},
			},
			AllowHiddenMapEntries: true,
		},
		"what columns should be shown (note that the name is always shown)",
		param.AltNames("show", "cols"),
	)

	ps.Add(paramNamesByLevel, psetter.Nil{},
		"just show the module names in level order",
		param.PostAction(paction.SetBool(&showHeader, false)),
		param.PostAction(paction.SetBool(&showIntro, false)),
		param.PostAction(paction.SetString(&sortBy, ColLevel)),
		param.PostAction(func(_ location.L, _ *param.ByName, _ []string) error {
			columnsToShow = map[string]bool{
				ColName: true,
			}
			return nil
		}),
	)

	ps.Add(paramFilter, psetter.Map{Value: &modFilter},
		"the module name to filter by."+
			" The report will only show this module"+
			" and any module that uses this module."+
			" The notion of 'used' is recursive so that"+
			" if the filter is on module A"+
			" and module B uses A and C uses B but not A (directly)"+
			" then modules A, B and C will be shown.",
	)

	// allow trailing arguments
	err := ps.SetNamedRemHandler(param.NullRemHandler{}, "go.mod-files")
	if err != nil {
		return err
	}

	return nil
}
