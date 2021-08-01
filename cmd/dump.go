// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"os"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

// DumpCmd is the kong `dump` command.
type DumpCmd struct {
	Order  types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`
	System string          `kong:"default='auto',enum='auto,dos3',help='DOS system used for image.'"`

	DiskImage string `kong:"arg,required,type='existingfile',help='Disk image to modify.'"`
	Filename  string `kong:"arg,required,help='Filename to use on disk.'"`
}

// Help displays extended help and examples.
func (d DumpCmd) Help() string {
	return `Examples:
	# Dump file GREMLINS on disk image games.dsk.
	diskii dump games.dsk GREMLINS`
}

// Run the `dump` command.
func (d *DumpCmd) Run(globals *types.Globals) error {
	op, _, err := disk.OpenFilename(d.DiskImage, d.Order, d.System, globals.DiskOperatorFactories, globals.Debug)
	if err != nil {
		return err
	}

	file, err := op.GetFile(d.Filename)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(file.Data)
	if err != nil {
		return err
	}
	return nil
}
