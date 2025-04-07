package main

import (
	"os"

	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/paramset"
	"github.com/nickwells/versionparams.mod/versionparams"
)

// makeParamSet generates the param set ready for parsing
func makeParamSet(prog *prog) *param.PSet {
	return paramset.NewOrPanic(
		versionparams.AddParams,
		addParams(prog),
		addExamples,
		SetGlobalConfigFile,
		SetConfigFile,
		param.SetProgramDescription("This will take a list of go.mod"+
			" files (or directories) as trailing arguments"+
			" (after '"+param.DfltTerminalParam+"'), parse them and print"+
			" a report. The report will show how they relate to one"+
			" another with regards to dependencies and can print them in"+
			" such an order that an earlier module does not depend on any"+
			" subsequent module."+
			"\n\n"+
			"By default any report will be preceded with a description of"+
			" what the various columns mean."+
			"\n\n"+
			"If a trailing argument does not end with "+
			"'"+string(os.PathSeparator)+"go.mod'"+
			" then it is taken as a directory name and the missing"+
			" filename is automatically appended."),
	)
}
