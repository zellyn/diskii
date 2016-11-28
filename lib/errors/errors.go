// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package errors contains helpers for creating and testing for
// certain types of errors.
package errors

import (
	"errors"
	"fmt"
)

// Copy of errors.New, so you this package can be imported instead.
func New(text string) error {
	return errors.New(text)
}

// --------------------- Out of space

// outOfSpace is an error that signals being out of space on a disk
// image.
type outOfSpace string

// OutOfSpaceI is the tag interface used to mark out of space errors.
type OutOfSpaceI interface {
	IsOutOfSpace()
}

var _ OutOfSpaceI = outOfSpace("test")

// Error returns the string message of an OutOfSpace error.
func (o outOfSpace) Error() string {
	return string(o)
}

// Tag method on our outOfSpace implementation.
func (o outOfSpace) IsOutOfSpace() {
}

// OutOfSpacef is fmt.Errorf for OutOfSpace errors.
func OutOfSpacef(format string, a ...interface{}) error {
	return outOfSpace(fmt.Sprintf(format, a...))
}

// IsOutOfSpace returns true if a given error is an OutOfSpace error.
func IsOutOfSpace(err error) bool {
	_, ok := err.(OutOfSpaceI)
	return ok
}

// --------------------- File exists

// fileExists is an error returned when a problem is caused by a file
// with the given name already existing.
type fileExists string

// FileExistsI is the tag interface used to mark FileExists errors.
type FileExistsI interface {
	IsFileExists()
}

var _ FileExistsI = fileExists("test")

// Error returns the string message of a FileExists error.
func (o fileExists) Error() string {
	return string(o)
}

// Tag method on our fileExists implementation.
func (o fileExists) IsFileExists() {
}

// FileExistsf is fmt.Errorf for FileExists errors.
func FileExistsf(format string, a ...interface{}) error {
	return fileExists(fmt.Sprintf(format, a...))
}

// IsFileExists returns true if a given error is a FileExists error.
func IsFileExists(err error) bool {
	_, ok := err.(FileExistsI)
	return ok
}

// --------------------- File not found

// fileNotFound is an error returned when a file with the given name
// cannot be found.
type fileNotFound string

// FileNotFoundI is the tag interface used to mark FileNotFound errors.
type FileNotFoundI interface {
	IsFileNotFound()
}

var _ FileNotFoundI = fileNotFound("test")

// Error returns the string message of a FileNotFound error.
func (o fileNotFound) Error() string {
	return string(o)
}

// Tag method on our fileNotFound implementation.
func (o fileNotFound) IsFileNotFound() {
}

// FileNotFoundf is fmt.Errorf for FileNotFound errors.
func FileNotFoundf(format string, a ...interface{}) error {
	return fileNotFound(fmt.Sprintf(format, a...))
}

// IsFileNotFound returns true if a given error is a FileNotFound error.
func IsFileNotFound(err error) bool {
	_, ok := err.(FileNotFoundI)
	return ok
}
