package disk

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/zellyn/diskii/types"
)

// OpenImage attempts to open an image on disk, using the provided ordering and system type.
func OpenImage(file *os.File, order string, system string, globals *types.Globals) (types.Operator, string, error) {
	bb, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	if len(bb) == FloppyDiskBytes {
		return openDoOrPo(bb, order, system, globals, strings.ToLower(path.Ext(file.Name())))
	}
	return nil, "", fmt.Errorf("OpenImage not implemented yet for non-disk-sized images")
}

func openDoOrPo(diskbytes []byte, order string, system string, globals *types.Globals, ext string) (types.Operator, string, error) {
	var factories []types.OperatorFactory
	for _, factory := range globals.DiskOperatorFactories {
		if system == "auto" || system == factory.Name() {
			factories = append(factories, factory)
		}
	}
	if len(factories) == 0 {
		return nil, "", fmt.Errorf("cannot find disk system with name %q", system)
	}
	orders := []string{order}
	switch order {
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
		return nil, "", fmt.Errorf("disk order %q invalid for %d-byte disk images", order, FloppyDiskBytes)
	}

	for _, order := range orders {
		swizzled, err := Swizzle(diskbytes, LogicalToPhysicalByName[order])
		if err != nil {
			return nil, "", err
		}
		for _, factory := range factories {
			if len(orders) == 1 && system != "auto" {
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
