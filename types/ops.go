// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// ops.go contains the interfaces and helper functions for operating
// on disk images logically: catalog, rename, delete, create files,
// etc.

package types

// Descriptor describes a file's characteristics.
type Descriptor struct {
	Name     string
	Fullname string // If there's a more complete filename (eg. Super-Mon), put it here.
	Sectors  int
	Blocks   int
	Length   int
	Locked   bool
	Type     Filetype
}

// OperatorFactory is the interface for getting operators, and finding out a bit
// about them before getting them.
type OperatorFactory interface {
	// Name returns the name of the operator.
	Name() string
	// SeemsToMatch returns true if the []byte disk image seems to match the
	// system of this operator.
	SeemsToMatch(diskbytes []byte, debug bool) bool
	// Operator returns an Operator for the []byte disk image.
	Operator(diskbytes []byte, debug bool) (Operator, error)
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
}

// FileInfo represents a file descriptor plus the content.
type FileInfo struct {
	Descriptor   Descriptor
	Data         []byte
	StartAddress uint16
}
