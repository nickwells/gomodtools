package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nickwells/check.mod/v2/check"
	"github.com/nickwells/dirsearch.mod/v2/dirsearch"
	"github.com/nickwells/location.mod/location"
)

// ModInfo records information gleaned from the go.mod files
type ModInfo struct {
	Loc              *location.L
	Name             string
	Reqs             []*ModInfo
	ReqCountInternal int
	ReqCountExternal int
	Level            int
	ReqdBy           []*ModInfo
	Packages         map[string]*PkgInfo
}

// NewModInfo creates a new ModInfo with the name populated and the Packages
// map initialised.
func NewModInfo(name string) *ModInfo {
	return &ModInfo{
		Name:     name,
		Packages: map[string]*PkgInfo{},
	}
}

type ModMap map[string]*ModInfo

// parseGoModFile parses the supplied file and uses the information found to
// populate the map of moduleInfo
func parseGoModFile(modules ModMap, f io.Reader, loc *location.L) *ModInfo {
	spacePattern := regexp.MustCompile("[ \t]+")

	scanner := bufio.NewScanner(f)
	var mi *ModInfo
	var inReqBlock bool

	for scanner.Scan() {
		loc.Incr()
		line := scanner.Text()
		switch line {
		case "":
			continue
		case "require (":
			inReqBlock = true
			continue
		case ")":
			inReqBlock = false
			continue
		}

		loc.SetContent(line)

		parts := spacePattern.Split(line, -1)
		switch parts[0] {
		case "module":
			mi = getModuleInfo(modules, parts, loc)
			if mi == nil {
				break
			}
		case "":
			if inReqBlock {
				mi.addReqs(modules, parts, loc)
			}
		case "require":
			mi.addReqs(modules, parts, loc)
		}
	}

	return mi
}

// getModuleInfo gets the module info for the named module. It will return
// nil if the module has already been defined (if it's seen a line starting
// with the word "module")
func getModuleInfo(modules ModMap, parts []string, loc *location.L) *ModInfo {
	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "Error: there is no module name at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : there are too few parts\n")
		fmt.Fprintf(os.Stderr, "     : a module name was expected\n")
		return nil
	}

	modName := parts[1]

	mi, ok := modules[modName]
	if !ok { // a new module so create it and add it to the map
		mi = NewModInfo(modName)
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
func (mi *ModInfo) addReqs(modules ModMap, parts []string, loc *location.L) {
	if mi == nil {
		fmt.Fprintf(os.Stderr, "Error: no module is defined at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : the module should be known\n")
		fmt.Fprintf(os.Stderr, "     : before requirements are stated\n")
		return
	}

	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "Error: there is no module name at %s\n", loc)
		fmt.Fprintf(os.Stderr, "     : there are too few parts\n")
		fmt.Fprintf(os.Stderr, "     : a required module name was expected\n")
		return
	}

	r := parts[1]
	reqdMI, ok := modules[r]
	if !ok { // the required module is not yet known, so create a new one
		reqdMI = NewModInfo(r)
		modules[r] = reqdMI
	}
	reqdMI.ReqdBy = append(reqdMI.ReqdBy, mi)
	mi.Reqs = append(mi.Reqs, reqdMI)
}

// calcLevel sets the level of the module to one greater than the max level
// of those modules that it requires. It will return true if the level has
// been changed.
func (mi *ModInfo) calcLevel() bool {
	levelChange := false
	for _, rmi := range mi.Reqs {
		if rmi.Level >= mi.Level {
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
func (mi *ModInfo) setReqCounts() {
	for _, rmi := range mi.Reqs {
		if rmi.Loc == nil {
			mi.ReqCountExternal++
		} else {
			mi.ReqCountInternal++
		}
	}
}

// getPackageInfo will walk the directory tree from the directory given and
// will gather statistics about the packages found.
func (mi *ModInfo) getPackageInfo(dirName string) {
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
			if pName == basePName {
				pkg.HasTestsInt = true
			} else {
				pkg.HasTestsAPI = true
			}
		} else {
			pkg.Files = append(pkg.Files, gi)
		}
	}
}

// GoInfo records Go information about a file
type GoInfo struct {
	FileName  string
	LineCount int
	Info      *ast.File
}

// PkgInfo records aggregate package information
type PkgInfo struct {
	Name        string
	ImportName  string
	Files       []GoInfo
	TestFiles   []GoInfo
	HasTestsInt bool
	HasTestsAPI bool
}

// getGoInfo finds Go information from the Go File
func getGoInfo(fileSet *token.FileSet, info *ast.File) GoInfo {
	file := fileSet.File(info.Pos())
	gi := GoInfo{
		FileName:  file.Name(),
		LineCount: file.LineCount(),
		Info:      info,
	}

	return gi
}
