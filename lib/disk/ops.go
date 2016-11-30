// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// ops.go contains the interfaces and helper functions for operating
// on disk images logically: catalog, rename, delete, create files,
// etc.

package disk

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Descriptor describes a file's characteristics.
type Descriptor struct {
	Name    string
	Sectors int
	Length  int
	Locked  bool
	Type    Filetype
}

// Operator is the interface that can operate on disks.
type Operator interface {
	// Name returns the name of the operator.
	Name() string
	// HasSubdirs returns true if the underlying operating system on the
	// disk allows subdirectories.
	HasSubdirs() bool
	// Catalog returns a catalog of disk entries. subdir should be empty
	// for operating systems that do not support subdirectories.
	Catalog(subdir string) ([]Descriptor, error)
	// GetFile retrieves a file by name.
	GetFile(filename string) (FileInfo, error)
	// Delete deletes a file by name. It returns true if the file was
	// deleted, false if it didn't exist.
	Delete(filename string) (bool, error)
	// PutFile writes a file by name. If the file exists and overwrite
	// is false, it returns with an error. Otherwise it returns true if
	// an existing file was overwritten.
	PutFile(filename string, fileInfo FileInfo, overwrite bool) (existed bool, err error)
}

// FileInfo represents a file descriptor plus the content.
type FileInfo struct {
	Descriptor   Descriptor
	Data         []byte
	StartAddress uint16
}

// operatorFactory is the type of functions that accept a SectorDisk,
// and may return an Operator interface to operate on it.
type operatorFactory func(SectorDisk) (Operator, error)

// operatorFactories is the map of currently-registered operator
// factories.
var operatorFactories map[string]operatorFactory

func init() {
	operatorFactories = make(map[string]operatorFactory)
}

// RegisterOperatorFactory registers an operator factory with the
// given name: a function that accepts a SectorDisk, and may return an
// Operator. It doesn't lock operatorFactories: it is expected to be
// called only from package `init` functions.
func RegisterOperatorFactory(name string, factory operatorFactory) {
	operatorFactories[name] = factory
}

// OperatorFor returns an Operator for the given SectorDisk, if possible.
func OperatorFor(sd SectorDisk) (Operator, error) {
	if len(operatorFactories) == 0 {
		return nil, errors.New("Cannot find an operator matching the given disk image (none registered)")
	}
	for _, factory := range operatorFactories {
		if operator, err := factory(sd); err == nil {
			return operator, nil
		}
	}
	names := make([]string, 0, len(operatorFactories))
	for name := range operatorFactories {
		names = append(names, `"`+name+`"`)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("Cannot find an operator matching the given disk image (tried %s)", strings.Join(names, ", "))
}
