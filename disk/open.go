package disk

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zellyn/diskii/types"
)

var diskOrdersByName map[string][]int = map[string][]int{
	"do":  Dos33LogicalToPhysicalSectorMap,
	"po":  ProDOSLogicalToPhysicalSectorMap,
	"raw": {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF},
}

// OpenImage attempts to open an image on disk, using the provided ordering and system type.
func OpenImage(file *os.File, globals *types.Globals) (types.Operator, string, error) {
	bb, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	if len(bb) == FloppyDiskBytes {
		return openDoOrPo(bb, globals, strings.ToLower(path.Ext(file.Name())))
	}
	return nil, "", fmt.Errorf("OpenImage not implemented yet for non-disk-sized images")
}

func openDoOrPo(diskbytes []byte, globals *types.Globals, ext string) (types.Operator, string, error) {
	var factories []types.OperatorFactory
	for _, factory := range globals.DiskOperatorFactories {
		if globals.System == "auto" || globals.System == factory.Name() {
			factories = append(factories, factory)
		}
	}
	if len(factories) == 0 {
		return nil, "", fmt.Errorf("cannot find disk system with name %q", globals.System)
	}
	orders := []string{globals.Order}
	switch globals.Order {
	case "do", "po":
		// nothing more
	case "auto":
		switch ext {
		case ".po":
			orders = []string{"po"}
		case ".do":
			orders = []string{"do"}
		case ".dsk", "":
			orders = []string{"do", "po"}
		default:
			return nil, "", fmt.Errorf("unknown disk image extension: %q", ext)
		}
	default:
		return nil, "", fmt.Errorf("disk order %q invalid for %d-byte disk images", globals.Order, FloppyDiskBytes)
	}

	for _, order := range orders {
		swizzled, err := Swizzle(diskbytes, diskOrdersByName[order])
		if err != nil {
			return nil, "", err
		}
		for _, factory := range factories {
			if len(orders) == 1 && globals.System != "auto" {
				if globals.Debug {
					fmt.Fprintf(os.Stderr, "Attempting to open with order=%s, system=%s.\n", order, factory.Name())
				}
				op, err := factory.Operator(swizzled, globals.Debug)
				if err != nil {
					return nil, "", err
				}
				return op, order, nil
			}

			if globals.Debug {
				fmt.Fprintf(os.Stderr, "Testing whether order=%s, system=%s seems to match.\n", order, factory.Name())
			}
			if factory.SeemsToMatch(swizzled, globals.Debug) {
				op, err := factory.Operator(swizzled, globals.Debug)
				if err == nil {
					return op, order, nil
				}
				if globals.Debug {
					fmt.Fprintf(os.Stderr, "Got error opening with order=%s, system=%s: %v\n", order, factory.Name(), err)
				}
			}
		}
	}
	return nil, "", fmt.Errorf("openDoOrPo not implemented yet")
}

// Swizzle changes the sector ordering according to the order parameter. If
// order is nil, it leaves the order unchanged.
func Swizzle(diskimage []byte, order []int) ([]byte, error) {
	if len(diskimage) != FloppyDiskBytes {
		return nil, fmt.Errorf("swizzling only works on disk images of %d bytes; got %d", FloppyDiskBytes, len(diskimage))
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
