// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

var shortnames bool // flag for whether to print short filenames
var debug bool

type LsCmd struct {
	ShortNames bool     `kong:"short='s',help='Whether to print short filenames (only makes a difference on Super-Mon disks).'"`
	Image      *os.File `kong:"arg,required,help='Disk/device image to read.'"`
	Directory  string   `kong:"arg,optional,help='Directory to list (ProDOS only).'"`
}

func (l *LsCmd) Run(globals *types.Globals) error {
	op, order, err := disk.OpenImage(l.Image, globals)
	if err != nil {
		return err
	}
	if globals.Debug {
		fmt.Fprintf(os.Stderr, "Opened disk with order %q, system %q\n", order, op.Name())
	}

	if l.Directory != "" {
		if !op.HasSubdirs() {
			return fmt.Errorf("Disks of type %q cannot have subdirectories", op.Name())
		}
	}
	fds, err := op.Catalog(l.Directory)
	if err != nil {
		return err
	}
	for _, fd := range fds {
		if !shortnames && fd.Fullname != "" {
			fmt.Println(fd.Fullname)
		} else {
			fmt.Println(fd.Name)
		}
	}
	return nil
}
