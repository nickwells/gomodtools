package main

import (
	"github.com/nickwells/location.mod/location"
	"github.com/nickwells/param.mod/v5/param"
	"github.com/nickwells/param.mod/v5/param/paction"
	"github.com/nickwells/param.mod/v5/param/psetter"
)

// addParams will add parameters to the passed param.PSet
func addParams(ps *param.PSet) error {
	ps.Add("hide-header", psetter.Bool{Value: &showHeader, Invert: true},
		"suppress the printing of the header",
		param.AltName("hide-hdr"),
		param.AltName("no-hdr"),
	)
	ps.Add("hide-intro", psetter.Bool{Value: &showIntro, Invert: true},
		"suppress the printing of the introductory text"+
			" explaining the meaning of the report",
		param.AltName("no-intro"),
	)
	ps.Add("brief", psetter.Nil{},
		"suppress the printing of both the introductory text and the headers",
		param.PostAction(paction.SetBool(&showHeader, false)),
		param.PostAction(paction.SetBool(&showIntro, false)),
	)

	ps.Add("hide-dup-levels", psetter.Bool{Value: &hideDupLevels},
		"suppress the printing of levels where the level value"+
			" is the same as on the previous line",
	)

	ps.Add("sort-order",
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
			}},
		"what order should the modules be sorted when reporting",
		param.AltName("sort-by"),
	)

	ps.Add("show-cols",
		psetter.EnumMap{
			Value: &columnsToShow,
			AllowedVals: psetter.AllowedVals{
				ColLevel:    "where the module lies in the dependency order",
				ColUseCount: "how heavily used the module is",
				ColUsesCountInt: "how much use the module makes" +
					" of other modules in the collection",
				ColUsesCountExt: "how much use the module makes" +
					" of modules not in the collection",
			},
			AllowHiddenMapEntries: true,
		},
		"what columns should be shown (note that the name is always shown)",
		param.AltName("show"),
		param.AltName("cols"),
	)

	ps.Add("names-by-level", psetter.Nil{},
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

	// allow trailing arguments
	err := ps.SetNamedRemHandler(param.NullRemHandler{}, "go.mod-files")
	if err != nil {
		return err
	}

	ps.AddExample(
		"gomodlayers -names-by-level"+
			" -- dir1/go.mod dir2/go.mod dir3/go.mod",
		"This will print just the names of the modules but in an order"+
			" such that no module depends on any of the modules listed"+
			" after it. This can be useful when you want to know the best"+
			" order to update the modules.")

	return nil
}
