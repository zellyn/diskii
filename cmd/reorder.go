package cmd

import (
	"fmt"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/helpers"
	"github.com/zellyn/diskii/types"
)

// ReorderCmd is the kong `reorder` command.
type ReorderCmd struct {
	Order     types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='Logical-to-physical sector order.'"`
	NewOrder  types.DiskOrder `kong:"default='auto',enum='auto,do,po',help='New Logical-to-physical sector order.'"`
	Overwrite bool            `kong:"short='f',help='Overwrite existing file?'"`

	DiskImage    string `kong:"arg,required,type='existingfile',help='Disk image to read.'"`
	NewDiskImage string `kong:"arg,optional,type='path',help='Disk image to write, if different.'"`
}

// Run the `reorder` command.
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

	overwrite := r.Overwrite
	filename := r.NewDiskImage
	if filename == "" {
		filename = r.DiskImage
		overwrite = true
	}
	return helpers.WriteOutput(filename, tobytes, overwrite)
}

// getOrders returns the input order, and the output order.
func getOrders(inFilename string, inOrder types.DiskOrder, outFilename string, outOrder types.DiskOrder) (types.DiskOrder, types.DiskOrder, error) {
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

	inGuess, outGuess := disk.OrderFromFilename(inFilename, types.DiskOrderUnknown), disk.OrderFromFilename(outFilename, types.DiskOrderUnknown)
	if inGuess == outGuess {
		if inGuess == types.DiskOrderUnknown {
			return "", "", fmt.Errorf("cannot determine input or output order from file extensions")
		}
		return "", "", fmt.Errorf("guessed order (%s) from file %q is the same as guessed order (%s) from file %q", inGuess, inFilename, outGuess, outFilename)
	}

	if inGuess == types.DiskOrderUnknown {
		return oppositeOrder(outGuess), outGuess, nil
	}
	if outGuess == types.DiskOrderUnknown {
		return inGuess, oppositeOrder(inGuess), nil
	}
	return inGuess, outGuess, nil
}

// oppositeOrder returns the opposite order from the input.
func oppositeOrder(order types.DiskOrder) types.DiskOrder {
	if order == types.DiskOrderDO {
		return types.DiskOrderPO
	}
	return types.DiskOrderDO
}
