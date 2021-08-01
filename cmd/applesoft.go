// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"os"

	"github.com/zellyn/diskii/basic"
	"github.com/zellyn/diskii/basic/applesoft"
	"github.com/zellyn/diskii/helpers"
	"github.com/zellyn/diskii/types"
)

type ApplesoftCmd struct {
	Decode DecodeCmd `kong:"cmd,help='Convert a binary Applesoft program to a text LISTing.'"`
}

type DecodeCmd struct {
	Filename string `kong:"arg,default='-',type='existingfile',help='Binary Applesoft file to read, or “-” for stdin.'"`

	Location uint16 `kong:"type='anybaseuint16',default='0x801',help='Starting program location in memory.'"`
	Raw      bool   `kong:"short='r',help='Print raw control codes (no escaping)'"`
}

func (d DecodeCmd) Help() string {
	return `Examples:
	# Dump the contents of HELLO and then decode it.
	diskii dump dos33master.dsk HELLO | diskii applesoft decode -`
}

func (d *DecodeCmd) Run(globals *types.Globals) error {
	contents, err := helpers.FileContentsOrStdIn(d.Filename)
	if err != nil {
		return err
	}
	listing, err := applesoft.Decode(contents, d.Location)
	if err != nil {
		return err
	}
	if d.Raw {
		os.Stdout.WriteString(listing.String())
	} else {
		os.Stdout.WriteString(basic.ChevronControlCodes(listing.String()))
	}
	return nil
}
