package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/dirsearch.mod/v2/dirsearch"
	"github.com/nickwells/location.mod/location"

	"golang.org/x/mod/modfile"
)

// modInfo records information gleaned from the go.mod files
type modInfo struct {
	Loc              *location.L
	Name             string
	DirectReqs       []*modInfo
	IndirectReqs     []*modInfo
	ReqCountInt      int
	ReqCountExt      int
	Level            int
	LinesOfCode      int
	ReqdByDirectly   []*modInfo
	ReqdByIndirectly []*modInfo
	Packages         map[string]*PkgInfo
}

// newModInfo creates a new ModInfo with the name populated and the Packages
// map initialised.
func newModInfo(name string) *modInfo {
	return &modInfo{
		Name:     name,
		Packages: map[string]*PkgInfo{},
	}
}

// sortCrossRefs sorts the cross reference entries by name
func (mi *modInfo) sortCrossRefs() {
	cmpFunc := func(a, b *modInfo) int {
		return strings.Compare(a.Name, b.Name)
	}
	slices.SortFunc(mi.ReqdByDirectly, cmpFunc)
	slices.SortFunc(mi.ReqdByIndirectly, cmpFunc)
	slices.SortFunc(mi.DirectReqs, cmpFunc)
	slices.SortFunc(mi.IndirectReqs, cmpFunc)
}

// parseGoModFile parses the supplied file and uses the information found to
// populate the map of moduleInfo
func parseGoModFile(modules modMap, contents []byte, loc *location.L) (
	*modInfo, error,
) {
	modFile, err := modfile.Parse(loc.Source(), contents, nil)
	if err != nil {
		return nil, err
	}

	mi := getModuleInfo(modules, modFile.Module.Mod.Path, loc)

	for _, req := range modFile.Require {
		mi.addReqs(modules, req.Mod.Path, req.Indirect)
	}

	return mi, nil
}

// getModuleInfo gets the module info for the named module. It will return
// nil if the module has already been defined (if it's seen a line starting
// with the word "module")
func getModuleInfo(modules modMap, modName string, loc *location.L) *modInfo {
	mi, ok := modules[modName]
	if !ok { // a new module so create it and add it to the map
		mi = newModInfo(modName)
		mi.Loc = loc

		modules[modName] = mi

		return mi
	}

	if mi.Loc == nil { // we've seen it used before but not defined
		mi.Loc = loc // so set the location of the module definition
		return mi
	}

	// Whoops: it's been defined before
	fmt.Fprintf(os.Stderr, "Error: module %s has been declared before", modName)
	fmt.Fprintf(os.Stderr, "     : firstly at %s\n", mi.Loc)
	fmt.Fprintf(os.Stderr, "     :     now at %s\n", loc)

	return nil
}

// addReqs expects to be passed a non-nil ModInfo and the parts
// of a require line. It will find the corresponding module for the required
// module and record that as a requirement of the module and also record that
// this module requires the other module. If there is a problem it will
// report it .
func (mi *modInfo) addReqs(modules modMap, requires string, indirect bool) {
	reqdMI, ok := modules[requires]
	if !ok { // the required module is not yet known, so create a new one
		reqdMI = newModInfo(requires)
		modules[requires] = reqdMI
	}

	if indirect {
		reqdMI.ReqdByIndirectly = append(reqdMI.ReqdByIndirectly, mi)
		mi.IndirectReqs = append(mi.IndirectReqs, reqdMI)

		return
	}

	reqdMI.ReqdByDirectly = append(reqdMI.ReqdByDirectly, mi)
	mi.DirectReqs = append(mi.DirectReqs, reqdMI)
}

// calcLevel sets the level of the module to one greater than the max level
// of those modules that it requires. It will return true if the level has
// been changed.
func (mi *modInfo) calcLevel() bool {
	levelChange := false

	for _, rmi := range mi.DirectReqs {
		if rmi.Level >= mi.Level &&
			rmi.Loc != nil { // ignore modules not in set of considered modules
			mi.Level = rmi.Level + 1
			levelChange = true
		}
	}

	return levelChange
}

// setReqCounts counts the number of internal and external requirements for
// the module. A required module is taken to be internal if it is in the set
// of modules being examined (and so the required module has a non-nil Loc
// field indicating that the module's go.mod file has been seen)
func (mi *modInfo) setReqCounts() {
	for _, rmi := range mi.DirectReqs {
		if rmi.Loc == nil {
			mi.ReqCountExt++
		} else {
			mi.ReqCountInt++
		}
	}
}

// getPackageInfo will walk the directory tree from the directory given and
// will gather statistics about the packages found.
func (mi *modInfo) getPackageInfo(dirName string) {
	dirName = filepath.Clean(dirName)

	// Note that Go ignores files and directories whose name begins with '.'
	// or '_' and directories named testdata
	fMap, errs := dirsearch.FindRecursePrune(dirName, -1,
		[]check.FileInfo{
			check.FileInfoName(
				check.Not(check.StringHasPrefix[string]("."), "hidden")),
			check.FileInfoName(
				check.Not(check.StringHasPrefix[string]("_"), "hidden")),
			check.FileInfoName(
				check.Not(check.ValEQ("testdata"), "testdata")),
		},
		check.FileInfoName(
			check.Not(check.StringHasPrefix[string]("."), "hidden")),
		check.FileInfoName(
			check.Not(check.StringHasPrefix[string]("_"), "hidden")),
		check.FileInfoName(check.StringHasSuffix[string](".go")))

	if len(errs) != 0 {
		fmt.Println("Errors found while finding the package Go files")

		for _, err := range errs {
			fmt.Println("\t", err)
		}

		return
	}

	fileSet := token.NewFileSet()
	for fName := range fMap {
		info, err := parser.ParseFile(fileSet, fName, nil, 0)
		if err != nil {
			fmt.Println("\t", err)
			continue
		}

		importName := filepath.Clean(
			mi.Name +
				filepath.Dir(
					strings.TrimPrefix(fName, dirName)))

		pName := info.Name.Name
		basePName := strings.TrimSuffix(pName, "_test")

		pkg, ok := mi.Packages[importName]
		if !ok {
			pkg = &PkgInfo{
				Name:       basePName,
				ImportName: importName,
			}
			mi.Packages[importName] = pkg
		}

		gi := getGoInfo(fileSet, info)

		if strings.HasSuffix(fName, "_test.go") {
			pkg.TestFiles = append(pkg.TestFiles, gi)
			pkg.TestFilesLoC += gi.LineCount

			if pName == basePName {
				pkg.HasTestsInt = true
			} else {
				pkg.HasTestsAPI = true
			}
		} else {
			pkg.Files = append(pkg.Files, gi)
			pkg.FilesLoC += gi.LineCount
			mi.LinesOfCode += gi.LineCount
		}
	}
}
