// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package supermon contains routines for working with the on-disk
// structures of NakedOS/Super-Mon disks.
package supermon

import (
	"fmt"

	"github.com/zellyn/diskii/lib/disk"
)

const (
	// FileIllegal (zero) is not allowed in the sector map.
	FileIllegal = 0
	// FileFree signifies unused space in the sector map.
	FileFree = 0xff
	// FileReserved signifies space used by NakedOS in the sector map.
	FileReserved = 0xfe
)

// SectorDiskShim is a shim to undo DOS 3.3 sector mapping:
// NakedOS/Super-Mon disks are typically represented as DOS 3.3 .dsk
// images, but NakedOS does no sector mapping.
type SectorDiskShim struct {
	Dos33 disk.SectorDisk
}

// ReadLogicalSector reads a single logical sector from the disk. It
// always returns 256 byes.
func (s SectorDiskShim) ReadLogicalSector(track byte, sector byte) ([]byte, error) {
	if sector >= 16 {
		return nil, fmt.Errorf("Expected sector between 0 and 15; got %d", sector)
	}
	sector = disk.Dos33PhysicalToLogicalSectorMap[int(sector)]
	return s.Dos33.ReadLogicalSector(track, sector)
}

// WriteLogicalSector writes a single logical sector to a disk. It
// expects exactly 256 bytes.
func (s SectorDiskShim) WriteLogicalSector(track byte, sector byte, data []byte) error {
	if sector >= 16 {
		return fmt.Errorf("Expected sector between 0 and 15; got %d", sector)
	}
	sector = disk.Dos33PhysicalToLogicalSectorMap[int(sector)]
	return s.Dos33.WriteLogicalSector(track, sector, data)
}

// SectorMap is the list of sectors by file. It's always 560 bytes
// long (35 tracks * 16 sectors).
type SectorMap []byte

// LoadSectorMap loads a NakedOS sector map.
func LoadSectorMap(sd disk.SectorDisk) (SectorMap, error) {
	sm := SectorMap(make([]byte, 560))
	sector09, err := sd.ReadLogicalSector(0, 9)
	if err != nil {
		return sm, err
	}
	sector0A, err := sd.ReadLogicalSector(0, 0xA)
	if err != nil {
		return sm, err
	}
	sector0B, err := sd.ReadLogicalSector(0, 0xB)
	if err != nil {
		return sm, err
	}
	copy(sm[0:0x30], sector09[0xd0:])
	copy(sm[0x30:0x130], sector0A)
	copy(sm[0x130:0x230], sector0B)
	return sm, nil
}

// FileForSector returns the file that owns the given track/sector.
func (sm SectorMap) FileForSector(track, sector byte) byte {
	if track >= 35 {
		panic(fmt.Sprintf("FileForSector called with track=%d > 34", track))
	}
	if sector >= 16 {
		panic(fmt.Sprintf("FileForSector called with sector=%d > 15", sector))
	}
	return sm[int(track)*16+int(sector)]
}

// SectorsForFile returns the list of sectors that belong to the given
// file.
func (sm SectorMap) SectorsForFile(file byte) []disk.TrackSector {
	var result []disk.TrackSector
	for track := byte(0); track < 35; track++ {
		for sector := byte(0); sector < 16; sector++ {
			if file == sm.FileForSector(track, sector) {
				result = append(result, disk.TrackSector{Track: track, Sector: sector})
			}
		}
	}
	return result
}

func (sm SectorMap) Verify() error {
	for sector := byte(0); sector <= 0xB; sector++ {
		if file := sm.FileForSector(0, sector); file != FileReserved {
			return fmt.Errorf("Expected track 0, sectors 0-C to be reserved (0xFE), but got 0x%02X in sector %X", file, sector)
		}
	}

	for track := byte(0); track < 35; track++ {
		for sector := byte(0); sector < 16; sector++ {
			file := sm.FileForSector(track, sector)
			if file == FileIllegal {
				return fmt.Errorf("Found illegal sector map value (%02X), in track %X sector %X", FileIllegal, track, sector)
			}
		}
	}

	return nil
}

// SectorsByFile returns a map of file number to slice of sectors.
func (sm SectorMap) SectorsByFile() map[byte][]disk.TrackSector {
	result := map[byte][]disk.TrackSector{}
	for file := byte(0x01); file < FileReserved; file++ {
		sectors := sm.SectorsForFile(file)
		if len(sectors) > 0 {
			result[file] = sectors
		}
	}
	return result
}
