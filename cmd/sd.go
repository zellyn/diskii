// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/helpers"
	"github.com/zellyn/diskii/types"
)

// SDCmd is the kong `mksd` command.
type SDCmd struct {
	Order types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`

	DiskImage string   `kong:"arg,required,type='path',help='Disk image to write.'"`
	Binary    *os.File `kong:"arg,required,help='Binary file to write to the disk.'"`

	Address uint16 `kong:"type='anybaseuint16',default='0x6000',help='Address to load the code at.'"`
	Start   uint16 `kong:"type='anybaseuint16',default='0xFFFF',help='Address to jump to. Defaults to 0xFFFF, which means “same as address flag”'"`
}

// Help displays extended help and examples.
func (s SDCmd) Help() string {
	return `
See https://github.com/peterferrie/standard-delivery for details.

Examples:
	# Load and run foo.o at the default address, then jump to the start of the loaded code.
	diskii mksd test.dsk foo.o

	# Load foo.o at address 0x2000, then jump to 0x2100.
	diskii mksd test.dsk foo.o --address 0x2000 --start 0x2100`
}

// Run the `mksd` command.
func (s *SDCmd) Run(globals *types.Globals) error {
	if s.Start == 0xFFFF {
		s.Start = s.Address
	}

	contents, err := io.ReadAll(s.Binary)
	if err != nil {
		return err
	}
	if s.Address%256 != 0 {
		return fmt.Errorf("address %d (%04X) not on a page boundary", s.Address, s.Address)
	}
	if s.Start < s.Address {
		return fmt.Errorf("start address %d (%04X) < load address %d (%04X)", s.Start, s.Start, s.Address, s.Address)
	}

	if int(s.Start) >= int(s.Address)+len(contents) {
		end := int(s.Address) + len(contents)
		return fmt.Errorf("start address %d (%04X) is beyond load address %d (%04X) + file length = %d (%04X)",
			s.Start, s.Start, s.Address, s.Address, end, end)
	}

	if int(s.Start)+len(contents) > 0xC000 {
		end := int(s.Start) + len(contents)
		return fmt.Errorf("start address %d (%04X) + file length %d (%04X) = %d (%04X), but we can't load past page 0xBF00",
			s.Start, s.Start, len(contents), len(contents), end, end)
	}

	sectors := (len(contents) + 255) / 256

	loader := []byte{
		0x01, 0xa8, 0xee, 0x06, 0x08, 0xad, 0x4e, 0x08, 0xc9, 0xc0, 0xf0, 0x40, 0x85, 0x27, 0xc8,
		0xc0, 0x10, 0x90, 0x09, 0xf0, 0x05, 0x20, 0x2f, 0x08, 0xa8, 0x2c, 0xa0, 0x01, 0x84, 0x3d,
		0xc8, 0xa5, 0x27, 0xf0, 0xdf, 0x8a, 0x4a, 0x4a, 0x4a, 0x4a, 0x09, 0xc0, 0x48, 0xa9, 0x5b,
		0x48, 0x60, 0xe6, 0x41, 0x06, 0x40, 0x20, 0x37, 0x08, 0x18, 0x20, 0x3c, 0x08, 0xe6, 0x40,
		0xa5, 0x40, 0x29, 0x03, 0x2a, 0x05, 0x2b, 0xa8, 0xb9, 0x80, 0xc0, 0xa9, 0x30, 0x4c, 0xa8,
		0xfc, 0x4c, byte(s.Start), byte(s.Start >> 8),
	}

	if len(loader)+sectors+1 > 256 {
		return fmt.Errorf("file %q is %d bytes long, max is %d", s.Binary.Name(), len(contents), (255-len(loader))*256)
	}

	for len(contents)%256 != 0 {
		contents = append(contents, 0)
	}

	diskbytes := make([]byte, disk.FloppyDiskBytes)

	var track, sector byte
	for i := 0; i < len(contents); i += 256 {
		sector += 2
		if sector >= disk.FloppySectors {
			sector = (disk.FloppySectors + 1) - sector
			if sector == 0 {
				track++
				if track >= disk.FloppyTracks {
					return fmt.Errorf("ran out of tracks")
				}
			}
		}

		address := int(s.Address) + i
		loader = append(loader, byte(address>>8))
		if err := disk.WriteSector(diskbytes, track, sector, contents[i:i+256]); err != nil {
			return err
		}
	}

	loader = append(loader, 0xC0)
	for len(loader) < 256 {
		loader = append(loader, 0)
	}

	if err := disk.WriteSector(diskbytes, 0, 0, loader); err != nil {
		return err
	}

	order := s.Order
	if order == types.DiskOrderAuto {
		order = disk.OrderFromFilename(s.DiskImage, types.DiskOrderDO)
	}
	rawBytes, err := disk.Swizzle(diskbytes, disk.PhysicalToLogicalByName[order])
	if err != nil {
		return err
	}
	return helpers.WriteOutput(s.DiskImage, rawBytes, true)
}
