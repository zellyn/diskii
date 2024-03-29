// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package disk contains routines for reading and writing various disk
// file formats.
package disk

import "github.com/zellyn/diskii/types"

// Various DOS33 disk characteristics.
const (
	FloppyTracks  = 35
	FloppySectors = 16 // Sectors per track
	// FloppyDiskBytes is the number of bytes on a DOS 3.3 disk.
	FloppyDiskBytes         = 143360              // 35 tracks * 16 sectors * 256 bytes
	FloppyTrackBytes        = 256 * FloppySectors // Bytes per track
	FloppyDiskBytes13Sector = 35 * 13 * 256
)

// Dos33LogicalToPhysicalSectorMap maps logical sector numbers to physical ones.
// See [UtA2 9-42 - Read Routines].
var Dos33LogicalToPhysicalSectorMap = []int{
	0x00, 0x0D, 0x0B, 0x09, 0x07, 0x05, 0x03, 0x01,
	0x0E, 0x0C, 0x0A, 0x08, 0x06, 0x04, 0x02, 0x0F,
}

// Dos33PhysicalToLogicalSectorMap maps physical sector numbers to logical ones.
// See [UtA2 9-42 - Read Routines].
var Dos33PhysicalToLogicalSectorMap = []int{
	0x00, 0x07, 0x0E, 0x06, 0x0D, 0x05, 0x0C, 0x04,
	0x0B, 0x03, 0x0A, 0x02, 0x09, 0x01, 0x08, 0x0F,
}

// ProDOSLogicalToPhysicalSectorMap maps logical sector numbers to pysical ones.
// See [UtA2e 9-43 - Sectors vs. Blocks].
var ProDOSLogicalToPhysicalSectorMap = []int{
	0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0E,
	0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F,
}

// ProDosPhysicalToLogicalSectorMap maps physical sector numbers to logical ones.
// See [UtA2e 9-43 - Sectors vs. Blocks].
var ProDosPhysicalToLogicalSectorMap = []int{
	0x00, 0x08, 0x01, 0x09, 0x02, 0x0A, 0x03, 0x0B,
	0x04, 0x0C, 0x05, 0x0D, 0x06, 0x0E, 0x07, 0x0F,
}

// LogicalToPhysicalByName maps from "do" and "po" to the corresponding
// logical-to-physical ordering.
var LogicalToPhysicalByName map[types.DiskOrder][]int = map[types.DiskOrder][]int{
	types.DiskOrderDO:  Dos33LogicalToPhysicalSectorMap,
	types.DiskOrderPO:  ProDOSLogicalToPhysicalSectorMap,
	types.DiskOrderRaw: {0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
}

// PhysicalToLogicalByName maps from "do" and "po" to the corresponding
// physical-to-logical ordering.
var PhysicalToLogicalByName map[types.DiskOrder][]int = map[types.DiskOrder][]int{
	types.DiskOrderDO:  Dos33PhysicalToLogicalSectorMap,
	types.DiskOrderPO:  ProDosPhysicalToLogicalSectorMap,
	types.DiskOrderRaw: {0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
}

// TrackSector is a pair of track/sector bytes.
type TrackSector struct {
	Track  byte
	Sector byte
}

// Block is a ProDOS block: 512 bytes.
type Block [512]byte
