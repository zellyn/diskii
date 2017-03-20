// This file contains the list of commands to run to re-generate
// generated files.

// Use go-bindata to embed static assets that we need.
//go:generate go-bindata -pkg data -prefix "data/" -o data/data.go data/disks data/boot
//go:generate goimports -w data/data.go

package main
