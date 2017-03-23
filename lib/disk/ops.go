// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// ops.go contains the interfaces and helper functions for operating
// on disk images logically: catalog, rename, delete, create files,
// etc.

package disk

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Descriptor describes a file's characteristics.
type Descriptor struct {
	Name     string
	Fullname string // If there's a more complete filename (eg. Super-Mon), put it here.
	Sectors  int
	Length   int
	Locked   bool
	Type     Filetype
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
	PutFile(fileInfo FileInfo, overwrite bool) (existed bool, err error)
	// Write writes the underlying disk to the given writer.
	Write(io.Writer) (int, error)
}

// FileInfo represents a file descriptor plus the content.
type FileInfo struct {
	Descriptor   Descriptor
	Data         []byte
	StartAddress uint16
}

// diskOperatorFactory is the type of functions that accept a SectorDisk,
// and may return an Operator interface to operate on it.
type diskOperatorFactory func(SectorDisk) (Operator, error)

// diskOperatorFactories is the map of currently-registered disk
// operator factories.
var diskOperatorFactories map[string]diskOperatorFactory

func init() {
	diskOperatorFactories = make(map[string]diskOperatorFactory)
}

// RegisterDiskOperatorFactory registers a disk operator factory with
// the given name: a function that accepts a SectorDisk, and may
// return an Operator. It doesn't lock diskOperatorFactories: it is
// expected to be called only from package `init` functions.
func RegisterDiskOperatorFactory(name string, factory diskOperatorFactory) {
	diskOperatorFactories[name] = factory
}

// OperatorForDisk returns an Operator for the given SectorDisk, if possible.
func OperatorForDisk(sd SectorDisk) (Operator, error) {
	if len(diskOperatorFactories) == 0 {
		return nil, errors.New("Cannot find an operator matching the given disk image (none registered)")
	}
	for _, factory := range diskOperatorFactories {
		if operator, err := factory(sd); err == nil {
			return operator, nil
		}
	}
	names := make([]string, 0, len(diskOperatorFactories))
	for name := range diskOperatorFactories {
		names = append(names, `"`+name+`"`)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("Cannot find a disk operator matching the given disk image (tried %s)", strings.Join(names, ", "))
}

// deviceOperatorFactory is the type of functions that accept a BlockDevice,
// and may return an Operator interface to operate on it.
type deviceOperatorFactory func(BlockDevice) (Operator, error)

// deviceOperatorFactories is the map of currently-registered device
// operator factories.
var deviceOperatorFactories map[string]deviceOperatorFactory

func init() {
	deviceOperatorFactories = make(map[string]deviceOperatorFactory)
}

// RegisterDeviceOperatorFactory registers a device operator factory with
// the given name: a function that accepts a BlockDevice, and may
// return an Operator. It doesn't lock deviceOperatorFactories: it is
// expected to be called only from package `init` functions.
func RegisterDeviceOperatorFactory(name string, factory deviceOperatorFactory) {
	deviceOperatorFactories[name] = factory
}

// OperatorForDevice returns an Operator for the given BlockDevice, if possible.
func OperatorForDevice(sd BlockDevice) (Operator, error) {
	if len(deviceOperatorFactories) == 0 {
		return nil, errors.New("Cannot find an operator matching the given device image (none registered)")
	}
	for _, factory := range deviceOperatorFactories {
		if operator, err := factory(sd); err == nil {
			return operator, nil
		}
	}
	names := make([]string, 0, len(deviceOperatorFactories))
	for name := range deviceOperatorFactories {
		names = append(names, `"`+name+`"`)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("Cannot find a device operator matching the given device image (tried %s)", strings.Join(names, ", "))
}
