package main

import (
	"github.com/nickwells/param.mod/v5/param"
)

// addExamples will add examples to the passed param.PSet
func addExamples(ps *param.PSet) error {
	ps.AddExample(
		"gomodlayers -names-by-level"+
			" -- dir1/go.mod dir2/go.mod dir3/go.mod",
		"This will print just the names of the modules but in an order"+
			" such that no module depends on any of the modules listed"+
			" after it. This can be useful when you want to know the best"+
			" order to update the modules.")
	ps.AddExample(
		"gomodlayers -- dir1/go.mod dir2/go.mod dir3/go.mod",
		"This will print the default output: an extensive introduction"+
			" explaining the results, column headings and then the"+
			" modules in an order such that no module depends on any of"+
			" the modules listed after it. The columns shown are the"+
			" module level, the full module name and how many of the"+
			" other modules use that module.")

	return nil
}
