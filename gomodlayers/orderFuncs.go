package main

// lessByLevel returns true or false according to the levels of the ModInfo
// entries. It will use the module name to resolve ties.
func lessByLevel(ms []*modInfo, i, j int) bool {
	if ms[i].Level < ms[j].Level {
		return true
	}

	if ms[i].Level > ms[j].Level {
		return false
	}

	return ms[i].Name < ms[j].Name
}

// lessByUseCount returns true or false according to the UseCounts of the
// ModInfo entries. It will use the module name to resolve ties.
func lessByUseCount(ms []*modInfo, i, j int) bool {
	if len(ms[i].ReqdBy) < len(ms[j].ReqdBy) {
		return true
	}

	if len(ms[i].ReqdBy) > len(ms[j].ReqdBy) {
		return false
	}

	return ms[i].Name < ms[j].Name
}

// lessByReqCountInt returns true or false according to the internal
// requirement count of the ModInfo entries. It will use the module name to
// resolve ties.
func lessByReqCountInt(ms []*modInfo, i, j int) bool {
	if ms[i].ReqCountInternal < ms[j].ReqCountInternal {
		return true
	}

	if ms[i].ReqCountInternal > ms[j].ReqCountInternal {
		return false
	}

	return ms[i].Name < ms[j].Name
}

// lessByReqCountExt returns true or false according to the external
// requirement count of the ModInfo entries. It will use the module name to
// resolve ties.
func lessByReqCountExt(ms []*modInfo, i, j int) bool {
	if ms[i].ReqCountExternal < ms[j].ReqCountExternal {
		return true
	}

	if ms[i].ReqCountExternal > ms[j].ReqCountExternal {
		return false
	}

	return ms[i].Name < ms[j].Name
}

// lessByPackages returns true or false according to the number of packages
// in the module. It will use the module name to resolve ties.
func lessByPackages(ms []*modInfo, i, j int) bool {
	if len(ms[i].Packages) < len(ms[j].Packages) {
		return true
	}

	if len(ms[i].Packages) > len(ms[j].Packages) {
		return false
	}

	return ms[i].Name < ms[j].Name
}

// lessByPkgLines returns true or false according to the number of lines in
// the packages in the module. It will use the module name to resolve ties.
func lessByPkgLines(ms []*modInfo, i, j int) bool {
	var locI, locJ int

	for _, pkg := range ms[i].Packages {
		for _, gi := range pkg.Files {
			locI += gi.LineCount
		}
	}

	for _, pkg := range ms[j].Packages {
		for _, gi := range pkg.Files {
			locJ += gi.LineCount
		}
	}

	if locI < locJ {
		return true
	}

	if locI > locJ {
		return false
	}

	return ms[i].Name < ms[j].Name
}
