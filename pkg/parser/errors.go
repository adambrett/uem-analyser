package parser

import "errors"

var (
	ErrMalformedRow     = errors.New("one of the rows could not be read as UEM data")
	ErrUnexpectedLabel  = errors.New("one of the rows uses an unexpected UEM label")
	ErrUnrecognizedFile = errors.New("this does not look like a UEM text export")
)
