package main

import (
	"go/ast"
	"go/token"
)

// GoInfo records Go information about a file
type GoInfo struct {
	FileName  string
	LineCount int
	Info      *ast.File
}

// PkgInfo records aggregate package information
type PkgInfo struct {
	Name         string
	ImportName   string
	Files        []GoInfo
	FilesLoC     int
	TestFiles    []GoInfo
	TestFilesLoC int
	HasTestsInt  bool
	HasTestsAPI  bool
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
