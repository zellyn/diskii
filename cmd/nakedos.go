// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
	"github.com/zellyn/diskii/lib/supermon"
)

// nakedosCmd represents the nakedos command
var nakedosCmd = &cobra.Command{
	Use:   "nakedos",
	Short: "work with NakedOS disks",
	Long: `diskii nakedos contains the subcommands useful for working
with NakedOS (and Super-Mon) disks`,
	Aliases: []string{"supermon"},
}

func init() {
	RootCmd.AddCommand(nakedosCmd)
}

// ----- mkhello command ----------------------------------------------------

var address uint16         // flag for address to load at
var start uint16           // flag for address to start execution at
const helloName = "FHELLO" // filename to use (if Super-Mon)

// mkhelloCmd represents the mkhello command
var mkhelloCmd = &cobra.Command{
	Use:   "mkhello <disk-image> filename",
	Short: "create an FHELLO program that loads and runs another file",
	Long: `
mkhello creates file DF01:FHELLO that loads and runs another program at a specific address.

Examples:
mkhello test.dsk FDEMO  # load and run FDEMO at the default address, then jump to the start of the loaded code.
mkhello test.dsk --address 0x2000 --start 0x2100 DF06  # load and run file DF06 at address 0x2000, and jump to 0x2100.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMkhello(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	nakedosCmd.AddCommand(mkhelloCmd)

	// Here you will define your flags and configuration settings.

	mkhelloCmd.Flags().Uint16VarP(&address, "address", "a", 0x6000, "memory location to load code at")
	mkhelloCmd.Flags().Uint16VarP(&start, "start", "s", 0x6000, "memory location to jump to")
}

// runMkhello performs the actual mkhello logic.
func runMkhello(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: diskii mkhello <disk image> <file-to-load>")
	}
	if address%256 != 0 {
		return fmt.Errorf("address %d (%04X) not on a page boundary", address, address)
	}
	if start < address {
		return fmt.Errorf("start address %d (%04X) < load address %d (%04X)", start, start, address, address)
	}
	op, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	if op.Name() != "nakedos" {
		return fmt.Errorf("mkhello only works on disks of type %q; got %q", "nakedos", op.Name())
	}
	nakOp, ok := op.(supermon.Operator)
	if !ok {
		return fmt.Errorf("internal error: cannot cast to expected supermon.Operator type")
	}
	addr, symbolAddr, _, err := nakOp.ST.FilesForCompoundName(args[1])
	if err != nil {
		return err
	}
	if addr == 0 && symbolAddr == 0 {
		return fmt.Errorf("cannot parse %q as valid filename", args[1])
	}
	toLoad := addr
	if addr == 0 {
		toLoad = symbolAddr
	}
	contents := []byte{
		0x20, 0x40, 0x03, // JSR  NAKEDOS
		0x6D, 0x01, 0xDC, // ADC  NKRDFILE
		0x2C, toLoad, 0xDF, // BIT ${file number to load}
		0x2C, 0x00, byte(address >> 8), // BIT ${target page}
		0xD8,                                // CLD
		0x4C, byte(start), byte(start >> 8), // JMP ${target page}
	}
	fileInfo := disk.FileInfo{
		Descriptor: disk.Descriptor{
			Name:   fmt.Sprintf("DF01:%s", helloName),
			Length: len(contents),
			Type:   disk.FiletypeBinary,
		},
		Data: contents,
	}
	_, err = op.PutFile(fileInfo, true)
	if err != nil {
		return err
	}
	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	_, err = op.Write(f)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
