package cmd

import (
	"fmt"
	"path"
	"strings"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/helpers"
	"github.com/zellyn/diskii/types"
)

type ReorderCmd struct {
	Order    string `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`
	NewOrder string `kong:"default='auto',enum='auto,do,po',help='New Logical-to-physical sector order.'"`
	Force    bool   `kong:"short='s',help='Overwrite existing file?'"`

	DiskImage    string `kong:"arg,required,type='existingfile',help='Disk image to read.'"`
	NewDiskImage string `kong:"arg,optional,type='path',help='Disk image to write, if different.'"`
}

func (r *ReorderCmd) Run(globals *types.Globals) error {
	fromOrderName, toOrderName, err := getOrders(r.DiskImage, r.Order, r.NewDiskImage, r.NewOrder)
	if err != nil {
		return err
	}
	frombytes, err := helpers.FileContentsOrStdIn(r.DiskImage)
	if err != nil {
		return err
	}
	fromOrder, ok := disk.LogicalToPhysicalByName[fromOrderName]
	if !ok {
		return fmt.Errorf("internal error: disk order '%s' not found", fromOrderName)
	}
	toOrder, ok := disk.PhysicalToLogicalByName[toOrderName]
	if !ok {
		return fmt.Errorf("internal error: disk order '%s' not found", toOrderName)
	}
	rawbytes, err := disk.Swizzle(frombytes, fromOrder)
	if err != nil {
		return err
	}
	tobytes, err := disk.Swizzle(rawbytes, toOrder)
	if err != nil {
		return err
	}
	return helpers.WriteOutput(r.NewDiskImage, tobytes, r.DiskImage, r.Force)
}

// getOrders returns the input order, and the output order.
func getOrders(inFilename string, inOrder string, outFilename string, outOrder string) (string, string, error) {
	if inOrder == "auto" && outOrder != "auto" {
		return oppositeOrder(outOrder), outOrder, nil
	}
	if outOrder == "auto" && inOrder != "auto" {
		return inOrder, oppositeOrder(inOrder), nil
	}
	if inOrder != outOrder {
		return inOrder, outOrder, nil
	}
	if inOrder != "auto" {
		return "", "", fmt.Errorf("identical order and new-order")
	}

	inGuess, outGuess := orderFromFilename(inFilename), orderFromFilename(outFilename)
	if inGuess == outGuess {
		if inGuess == "" {
			return "", "", fmt.Errorf("cannot determine input or output order from file extensions")
		}
		return "", "", fmt.Errorf("guessed order (%s) from file %q is the same as guessed order (%s) from file %q", inGuess, inFilename, outGuess, outFilename)
	}

	if inGuess == "" {
		return oppositeOrder(outGuess), outGuess, nil
	}
	if outGuess == "" {
		return inGuess, oppositeOrder(inGuess), nil
	}
	return inGuess, outGuess, nil
}

// oppositeOrder returns the opposite order from the input.
func oppositeOrder(order string) string {
	if order == "do" {
		return "po"
	}
	return "do"
}

// orderFromFilename tries to guess the disk order from the filename, using the extension.
func orderFromFilename(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".dsk", ".do":
		return "do"
	case ".po":
		return "po"
	default:
		return ""
	}
}
