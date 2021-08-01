package disk

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zellyn/diskii/helpers"
	"github.com/zellyn/diskii/types"
)

// OpenFilename attempts to open a disk or device image, using the provided ordering and system type.
func OpenFilename(filename string, order types.DiskOrder, system string, operatorFactories []types.OperatorFactory, debug bool) (types.Operator, types.DiskOrder, error) {
	if filename == "-" {
		return OpenFile(os.Stdin, order, system, operatorFactories, debug)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	return OpenFile(file, order, system, operatorFactories, debug)
}

// OpenImage attempts to open a disk or device image, using the provided ordering and system type.
// OpenImage will close the file.
func OpenFile(file *os.File, order types.DiskOrder, system string, operatorFactories []types.OperatorFactory, debug bool) (types.Operator, types.DiskOrder, error) {
	bb, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	if err := file.Close(); err != nil {
		return nil, "", err
	}
	return OpenImage(bb, file.Name(), order, system, operatorFactories, debug)
}

// OpenImage attempts to open a disk or device image, using the provided ordering and system type.
func OpenImage(filebytes []byte, filename string, order types.DiskOrder, system string, operatorFactories []types.OperatorFactory, debug bool) (types.Operator, types.DiskOrder, error) {
	ext := strings.ToLower(path.Ext(filename))
	size := len(filebytes)
	if size == FloppyDiskBytes {
		return openDoOrPo(filebytes, order, system, ext, operatorFactories, debug)
	}
	if size == FloppyDiskBytes13Sector {
		return nil, "", fmt.Errorf("cannot open 13-sector disk images (yet)")
	}

	if ext == ".hdv" {
		return openHDV(filebytes, order, system, operatorFactories, debug)
	}
	return nil, "", fmt.Errorf("can only open disk-sized images and .hdv files")
}

func openHDV(rawbytes []byte, order types.DiskOrder, system string, operatorFactories []types.OperatorFactory, debug bool) (types.Operator, types.DiskOrder, error) {
	size := len(rawbytes)
	if size%512 > 0 {
		return nil, "", fmt.Errorf("can only open .hdv files that are a multiple of 512 bytes: %d %% 512 == %d", size, size%512)
	}
	if size/512 > 65536 {
		return nil, "", fmt.Errorf("can only open .hdv up to size 32MiB (%d); got %d", 65536*512, size)
	}
	if order != "auto" && order != types.DiskOrderPO {
		return nil, "", fmt.Errorf("cannot open .hdv file in %q order", order)
	}
	if system != "auto" && system != "prodos" {
		return nil, "", fmt.Errorf("cannot open .hdv file with %q system", system)
	}
	for _, factory := range operatorFactories {
		if factory.Name() == "prodos" {
			op, err := factory.Operator(rawbytes, debug)
			if err != nil {
				return nil, "", err
			}
			return op, types.DiskOrderPO, nil
		}
	}
	return nil, "", fmt.Errorf("unable to find prodos module to open .hdv file") // Should not happen.
}

func openDoOrPo(rawbytes []byte, order types.DiskOrder, system string, ext string, operatorFactories []types.OperatorFactory, debug bool) (types.Operator, types.DiskOrder, error) {
	var factories []types.OperatorFactory
	for _, factory := range operatorFactories {
		if system == "auto" || system == factory.Name() {
			factories = append(factories, factory)
		}
	}
	if len(factories) == 0 {
		return nil, "", fmt.Errorf("cannot find disk system with name %q", system)
	}
	orders := []types.DiskOrder{order}
	switch order {
	case types.DiskOrderDO, types.DiskOrderPO:
		// nothing more
	case types.DiskOrderAuto:
		switch ext {
		case ".po":
			orders = []types.DiskOrder{types.DiskOrderPO}
		case ".do":
			orders = []types.DiskOrder{types.DiskOrderDO}
		case ".dsk", "":
			orders = []types.DiskOrder{types.DiskOrderDO, types.DiskOrderPO}
		default:
			return nil, "", fmt.Errorf("unknown disk image extension: %q", ext)
		}
	default:
		return nil, "", fmt.Errorf("disk order %q invalid for %d-byte disk images", order, FloppyDiskBytes)
	}

	for _, order := range orders {
		swizzled, err := Swizzle(rawbytes, LogicalToPhysicalByName[order])
		if err != nil {
			return nil, "", err
		}
		for _, factory := range factories {
			diskbytes, err := Swizzle(swizzled, PhysicalToLogicalByName[factory.DiskOrder()])
			if err != nil {
				return nil, "", err
			}

			if len(orders) == 1 && system != "auto" {
				if debug {
					fmt.Fprintf(os.Stderr, "Attempting to open with order=%s, system=%s.\n", order, factory.Name())
				}
				op, err := factory.Operator(diskbytes, debug)
				if err != nil {
					return nil, "", err
				}
				return op, order, nil
			}

			if debug {
				fmt.Fprintf(os.Stderr, "Testing whether order=%s, system=%s seems to match.\n", order, factory.Name())
			}
			if factory.SeemsToMatch(diskbytes, debug) {
				op, err := factory.Operator(diskbytes, debug)
				if err == nil {
					return op, order, nil
				}
				if debug {
					fmt.Fprintf(os.Stderr, "Got error opening with order=%s, system=%s: %v\n", order, factory.Name(), err)
				}
			}
		}
	}
	return nil, "", fmt.Errorf("unabled to open disk image")
}

// Swizzle changes the sector ordering according to the order parameter. If
// order is nil, it leaves the order unchanged.
func Swizzle(diskimage []byte, order []int) ([]byte, error) {
	if len(diskimage) != FloppyDiskBytes {
		return nil, fmt.Errorf("reordering only works on disk images of %d bytes; got %d", FloppyDiskBytes, len(diskimage))
	}
	if err := validateOrder(order); err != nil {
		return nil, fmt.Errorf("called Swizzle with weird order: %w", err)
	}

	result := make([]byte, FloppyDiskBytes)
	for track := 0; track < FloppyTracks; track++ {
		for sector := 0; sector < FloppySectors; sector++ {
			data, err := ReadSector(diskimage, byte(track), byte(sector))
			if err != nil {
				return nil, err
			}
			err = WriteSector(result, byte(track), byte(order[sector]), data)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func UnSwizzle(diskimage []byte, order []int) ([]byte, error) {
	if err := validateOrder(order); err != nil {
		return nil, fmt.Errorf("called UnSwizzle with weird order: %w", err)
	}
	reverseOrder := make([]int, FloppySectors)
	for index, mapping := range order {
		reverseOrder[mapping] = index
	}
	return Swizzle(diskimage, reverseOrder)
}

// validateOrder validates that an order mapping is valid, and maps [0,15] onto
// [0,15] without repeats.
func validateOrder(order []int) error {
	if len(order) != FloppySectors {
		return fmt.Errorf("len=%d; want %d: %v", len(order), FloppySectors, order)
	}
	seen := make(map[int]bool)
	for i, mapping := range order {
		if mapping < 0 || mapping > 15 {
			return fmt.Errorf("mapping %d:%d is not in [0,15]: %v", i, mapping, order)
		}
		if seen[mapping] {
			return fmt.Errorf("mapping %d:%d is a repeat: %v", i, mapping, order)
		}
		seen[mapping] = true
	}
	return nil
}

// OrderFromFilename tries to guess the disk order from the filename, using the extension.
func OrderFromFilename(filename string, defaultOrder types.DiskOrder) types.DiskOrder {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".dsk", ".do":
		return types.DiskOrderDO
	case ".po":
		return types.DiskOrderPO
	default:
		return defaultOrder
	}
}

// WriteBack writes a disk image back out.
func WriteBack(filename string, op types.Operator, diskFileOrder types.DiskOrder, overwrite bool) error {
	logicalBytes := op.GetBytes()
	// If it's not floppy-sized, we don't swizzle at all.
	if len(logicalBytes) != FloppyDiskBytes {
		return helpers.WriteOutput(filename, logicalBytes, overwrite)
	}

	// Go from logical sectors for the operator back to physical sectors.
	physicalBytes, err := Swizzle(logicalBytes, LogicalToPhysicalByName[op.DiskOrder()])
	if err != nil {
		return err
	}

	// Go from physical sectors to the disk order (DO or PO)
	diskBytes, err := Swizzle(physicalBytes, PhysicalToLogicalByName[diskFileOrder])
	if err != nil {
		return err
	}

	return helpers.WriteOutput(filename, diskBytes, overwrite)
}
