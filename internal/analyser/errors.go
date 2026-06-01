package analyser

import (
	"errors"
	"fmt"
)

var (
	ErrFileTooLarge = errors.New("this file is over the 5MB limit")
	ErrNoFiles      = errors.New("please choose at least one UEM text file")
)

type FileError struct {
	Name string
	Err  error
}

func (e FileError) Error() string {
	return fmt.Sprintf("%s: %v", e.Name, e.Err)
}

func (e FileError) Unwrap() error {
	return e.Err
}
