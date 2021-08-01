// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

type LsCmd struct {
	Order  types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`
	System string          `kong:"default='auto',enum='auto,dos3,prodos,nakedos',help='DOS system used for image.'"`

	ShortNames bool     `kong:"short='s',help='Whether to print short filenames (only makes a difference on Super-Mon disks).'"`
	Image      *os.File `kong:"arg,required,help='Disk/device image to read.'"`
	Directory  string   `kong:"arg,optional,help='Directory to list (ProDOS only).'"`
}

func (l LsCmd) Help() string {
	return `Examples:
	# Simple ls of a disk image
	diskii ls games.dsk
	# Get really explicit about disk order and system
	diskii ls --order do --system nakedos Super-Mon-2.0.dsk`
}

func (l *LsCmd) Run(globals *types.Globals) error {
	op, order, err := disk.OpenFile(l.Image, l.Order, l.System, globals.DiskOperatorFactories, globals.Debug)
	if err != nil {
		return fmt.Errorf("%w: %s", err, l.Image.Name())
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
		if !l.ShortNames && fd.Fullname != "" {
			fmt.Println(fd.Fullname)
		} else {
			fmt.Println(fd.Name)
		}
	}
	return nil
}
