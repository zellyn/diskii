// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

type DeleteCmd struct {
	Order     types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`
	System    string          `kong:"default='auto',enum='auto,dos3',help='DOS system used for image.'"`
	MissingOk bool            `kong:"short='f',help='Overwrite existing file?'"`

	DiskImage string `kong:"arg,required,type='existingfile',help='Disk image to modify.'"`
	Filename  string `kong:"arg,required,help='Filename to use on disk.'"`
}

func (d DeleteCmd) Help() string {
	return `Examples:
	# Delete file GREMLINS on disk image games.dsk.
	diskii rm games.dsk GREMLINS`
}

func (d *DeleteCmd) Run(globals *types.Globals) error {
	op, order, err := disk.OpenFilename(d.DiskImage, d.Order, d.System, globals.DiskOperatorFactories, globals.Debug)
	if err != nil {
		return err
	}

	deleted, err := op.Delete(d.Filename)
	if err != nil {
		return err
	}
	if !deleted && !d.MissingOk {
		return fmt.Errorf("file %q not found (use -f to prevent this being an error)", d.Filename)
	}
	return disk.WriteBack(d.DiskImage, op, order, true)
}
