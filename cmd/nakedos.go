// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/supermon"
	"github.com/zellyn/diskii/types"
)

const helloName = "FHELLO" // filename to use (if Super-Mon)

type NakedOSCmd struct {
	Mkhello MkHelloCmd `kong:"cmd,help='Create an FHELLO program that loads and runs another file.'"`
}

// Help shows extended help on NakedOS/Super-Mon.
func (n NakedOSCmd) Help() string {
	return `NakedOS and Super-Mon were created by the amazing Martin Haye. For more information see:
	Source/docs: https://bitbucket.org/martin.haye/super-mon/
	Presentation: https://www.kansasfest.org/2012/08/2010-haye-nakedos/`
}

type MkHelloCmd struct {
	Order types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`

	DiskImage string `kong:"arg,required,type='existingfile',help='Disk image to modify.'"`
	Filename  string `kong:"arg,required,help='Name of NakedOS file to load.'"`

	Address uint16 `kong:"type='anybaseuint16',default='0x6000',help='Address to load the code at.'"`
	Start   uint16 `kong:"type='anybaseuint16',default='0xFFFF',help='Address to jump to. Defaults to 0xFFFF, which means “same as address flag”'"`
}

func (m MkHelloCmd) Help() string {
	return `This command creates a very short DF01:FHELLO program that simply loads another program of your choice.
	
Examples:
	# Load and run FDEMO at the default address, then jump to the start of the loaded code.
	mkhello test.dsk FDEMO

	# Load and run file DF06 at address 0x2000, and jump to 0x2100.
	mkhello test.dsk --address 0x2000 --start 0x2100 DF06`
}

func (m *MkHelloCmd) Run(globals *types.Globals) error {
	if m.Start == 0xFFFF {
		m.Start = m.Address
	}

	if m.Address%256 != 0 {
		return fmt.Errorf("address %d (%04X) not on a page boundary", m.Address, m.Address)
	}
	if m.Start < m.Address {
		return fmt.Errorf("start address %d (%04X) < load address %d (%04X)", m.Start, m.Start, m.Address, m.Address)
	}

	op, order, err := disk.OpenFilename(m.DiskImage, m.Order, "auto", globals.DiskOperatorFactories, globals.Debug)
	if err != nil {
		return err
	}

	if op.Name() != "nakedos" {
		return fmt.Errorf("mkhello only works on disks of type %q; got %q", "nakedos", op.Name())
	}
	nakOp, ok := op.(supermon.Operator)
	if !ok {
		return fmt.Errorf("internal error: cannot cast to expected supermon.Operator type (got %T)", op)
	}
	addr, symbolAddr, _, err := nakOp.ST.FilesForCompoundName(m.Filename)
	if err != nil {
		return err
	}
	if addr == 0 && symbolAddr == 0 {
		return fmt.Errorf("cannot parse %q as valid filename", m.Filename)
	}
	toLoad := addr
	if addr == 0 {
		toLoad = symbolAddr
	}
	contents := []byte{
		0x20, 0x40, 0x03, // JSR  NAKEDOS
		0x6D, 0x01, 0xDC, // ADC  NKRDFILE
		0x2C, toLoad, 0xDF, // BIT ${file number to load}
		0x2C, 0x00, byte(m.Address >> 8), // BIT ${target page}
		0xD8,                                    // CLD
		0x4C, byte(m.Start), byte(m.Start >> 8), // JMP ${target page}
	}
	fileInfo := types.FileInfo{
		Descriptor: types.Descriptor{
			Name:   fmt.Sprintf("DF01:%s", helloName),
			Length: len(contents),
			Type:   types.FiletypeBinary,
		},
		Data: contents,
	}
	_ = fileInfo

	_, err = op.PutFile(fileInfo, true)
	if err != nil {
		return err
	}

	return disk.WriteBack(m.DiskImage, op, order, true)
}
