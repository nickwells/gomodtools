package main

// lessByLevel returns true or false according to the levels of the ModInfo
// entries. It will use the module name to resolve ties.
func lessByLevel(ms []*ModInfo, i, j int) bool {
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
func lessByUseCount(ms []*ModInfo, i, j int) bool {
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
func lessByReqCountInt(ms []*ModInfo, i, j int) bool {
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
func lessByReqCountExt(ms []*ModInfo, i, j int) bool {
	if ms[i].ReqCountExternal < ms[j].ReqCountExternal {
		return true
	}

	if ms[i].ReqCountExternal > ms[j].ReqCountExternal {
		return false
	}

	return ms[i].Name < ms[j].Name
}
