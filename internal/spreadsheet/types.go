package spreadsheet

import "github.com/adambrett/uem-analyser/pkg/parser"

type Workbook struct {
	Participant string
	Sessions    []SessionFile
}

type SessionFile struct {
	Name    string
	Session parser.Session
}
