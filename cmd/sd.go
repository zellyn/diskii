// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
	"github.com/zellyn/diskii/lib/helpers"
)

var sdAddress uint16 // flag for address to load at
var sdStart uint16   // flag for address to start execution at

// mksdCmd represents the mksd command
var mksdCmd = &cobra.Command{
	Use:   "mksd",
	Short: "create a Standard-Delivery disk image",
	Long: `diskii mksd creates a "Standard Delivery" disk image containing a binary.
See https://github.com/peterferrie/standard-delivery for details.

Examples:
mksd test.dsk foo.o  # load and run foo.o at the default address, then jump to the start of the loaded code.
mksd test.dsk foo.o --address 0x2000 --start 0x2100  # load foo.o at address 0x2000, then jump to 0x2100.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMkSd(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(mksdCmd)
	mksdCmd.Flags().Uint16VarP(&sdAddress, "address", "a", 0x6000, "memory location to load code at")
	mksdCmd.Flags().Uint16VarP(&sdStart, "start", "s", 0x6000, "memory location to jump to")
}

// ----- mksd command -------------------------------------------------------

// runMkSd performs the actual mksd logic.
func runMkSd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: diskii mksd <disk image> <file-to-load>")
	}
	contents, err := helpers.FileContentsOrStdIn(args[1])
	if err != nil {
		return err
	}
	if sdAddress%256 != 0 {
		return fmt.Errorf("address %d (%04X) not on a page boundary", sdAddress, sdAddress)
	}
	if sdStart < sdAddress {
		return fmt.Errorf("start address %d (%04X) < load address %d (%04X)", sdStart, sdStart, sdAddress, sdAddress)
	}

	if int(sdStart) >= int(sdAddress)+len(contents) {
		end := int(sdAddress) + len(contents)
		return fmt.Errorf("start address %d (%04X) is beyond load address %d (%04X) + file length = %d (%04X)",
			sdStart, sdStart, sdAddress, sdAddress, end, end)
	}

	if int(sdStart)+len(contents) > 0xC000 {
		end := int(sdStart) + len(contents)
		return fmt.Errorf("start address %d (%04X) + file length %d (%04X) = %d (%04X), but we can't load past page 0xBF00",
			sdStart, sdStart, len(contents), len(contents), end, end)
	}

	sectors := (len(contents) + 255) / 256

	loader := []byte{
		0x01, 0xa8, 0xee, 0x06, 0x08, 0xad, 0x4e, 0x08, 0xc9, 0xc0, 0xf0, 0x40, 0x85, 0x27, 0xc8,
		0xc0, 0x10, 0x90, 0x09, 0xf0, 0x05, 0x20, 0x2f, 0x08, 0xa8, 0x2c, 0xa0, 0x01, 0x84, 0x3d,
		0xc8, 0xa5, 0x27, 0xf0, 0xdf, 0x8a, 0x4a, 0x4a, 0x4a, 0x4a, 0x09, 0xc0, 0x48, 0xa9, 0x5b,
		0x48, 0x60, 0xe6, 0x41, 0x06, 0x40, 0x20, 0x37, 0x08, 0x18, 0x20, 0x3c, 0x08, 0xe6, 0x40,
		0xa5, 0x40, 0x29, 0x03, 0x2a, 0x05, 0x2b, 0xa8, 0xb9, 0x80, 0xc0, 0xa9, 0x30, 0x4c, 0xa8,
		0xfc, 0x4c, byte(sdStart), byte(sdStart >> 8),
	}

	if len(loader)+sectors+1 > 256 {
		return fmt.Errorf("file %q is %d bytes long, max is %d", args[1], len(contents), (255-len(loader))*256)
	}

	for len(contents)%256 != 0 {
		contents = append(contents, 0)
	}

	sd := disk.Empty()

	var track, sector byte
	for i := 0; i < len(contents); i += 256 {
		sector += 2
		if sector >= sd.Sectors() {
			sector = (sd.Sectors() + 1) - sector
			if sector == 0 {
				track++
				if track >= sd.Tracks() {
					return fmt.Errorf("ran out of tracks")
				}
			}
		}

		address := int(sdAddress) + i
		loader = append(loader, byte(address>>8))
		if err := sd.WritePhysicalSector(track, sector, contents[i:i+256]); err != nil {
			return err
		}
	}

	loader = append(loader, 0xC0)
	for len(loader) < 256 {
		loader = append(loader, 0)
	}

	if err := sd.WritePhysicalSector(0, 0, loader); err != nil {
		return err
	}

	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	_, err = sd.Write(f)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
