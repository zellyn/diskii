// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package disk contains routines for reading and writing various disk
// file formats.
package disk

import (
	"fmt"
	"io"
	"path"
	"strings"
)

// Various DOS33 disk characteristics.
const (
	DOS33Tracks  = 35
	DOS33Sectors = 16 // Sectors per track
	// DOS33DiskBytes is the number of bytes on a DOS 3.3 disk.
	DOS33DiskBytes  = 143360             // 35 tracks * 16 sectors * 256 bytes
	DOS33TrackBytes = 256 * DOS33Sectors // Bytes per track
)

// Dos33LogicalToPhysicalSectorMap maps logical sector numbers to physical ones.
// See [UtA2 9-42 - Read Routines].
var Dos33LogicalToPhysicalSectorMap = []byte{
	0x00, 0x0D, 0x0B, 0x09, 0x07, 0x05, 0x03, 0x01,
	0x0E, 0x0C, 0x0A, 0x08, 0x06, 0x04, 0x02, 0x0F,
}

// Dos33PhysicalToLogicalSectorMap maps physical sector numbers to logical ones.
// See [UtA2 9-42 - Read Routines].
var Dos33PhysicalToLogicalSectorMap = []byte{
	0x00, 0x07, 0x0E, 0x06, 0x0D, 0x05, 0x0C, 0x04,
	0x0B, 0x03, 0x0A, 0x02, 0x09, 0x01, 0x08, 0x0F,
}

// TrackSector is a pair of track/sector bytes.
type TrackSector struct {
	Track  byte
	Sector byte
}

// SectorDisk is the interface use to read and write disks by physical
// (matches sector header) sector number.
type SectorDisk interface {
	// ReadPhysicalSector reads a single physical sector from the disk. It
	// always returns 256 byes.
	ReadPhysicalSector(track byte, sector byte) ([]byte, error)
	// WritePhysicalSector writes a single physical sector to a disk. It
	// expects exactly 256 bytes.
	WritePhysicalSector(track byte, sector byte, data []byte) error
	// Sectors returns the number of sectors on the SectorDisk
	Sectors() byte
	// Tracks returns the number of tracks on the SectorDisk
	Tracks() byte
	// Write writes the disk contents to the given file.
	Write(io.Writer) (int, error)
}

// LogicalSectorDisk is the interface used to read and write a disk by
// *logical* sector number.
type LogicalSectorDisk interface {
	// ReadLogicalSector reads a single logical sector from the disk. It
	// always returns 256 byes.
	ReadLogicalSector(track byte, sector byte) ([]byte, error)
	// WriteLogicalSector writes a single logical sector to a disk. It
	// expects exactly 256 bytes.
	WriteLogicalSector(track byte, sector byte, data []byte) error
	// Sectors returns the number of sectors on the SectorDisk
	Sectors() byte
	// Tracks returns the number of tracks on the SectorDisk
	Tracks() byte
	// Write writes the disk contents to the given file.
	Write(io.Writer) (int, error)
}

// MappedDisk wraps a SectorDisk as a LogicalSectorDisk, handling the
// logical-to-physical sector mapping.
type MappedDisk struct {
	sectorDisk        SectorDisk // The underlying physical sector disk.
	logicalToPhysical []byte     // The mapping of logical to physical sectors.
}

var _ LogicalSectorDisk = MappedDisk{}

// NewMappedDisk returns a MappedDisk with the given
// logical-to-physical sector mapping.
func NewMappedDisk(sd SectorDisk, logicalToPhysical []byte) (MappedDisk, error) {
	if logicalToPhysical != nil && len(logicalToPhysical) != int(sd.Sectors()) {
		return MappedDisk{}, fmt.Errorf("NewMappedDisk called on a disk image with %d sectors per track, but a mapping of length %d", sd.Sectors(), len(logicalToPhysical))
	}
	if logicalToPhysical == nil {
		logicalToPhysical = make([]byte, int(sd.Sectors()))
		for i := range logicalToPhysical {
			logicalToPhysical[i] = byte(i)
		}
	}
	return MappedDisk{
		sectorDisk:        sd,
		logicalToPhysical: logicalToPhysical,
	}, nil
}

// ReadLogicalSector reads a single logical sector from the disk. It
// always returns 256 byes.
func (md MappedDisk) ReadLogicalSector(track byte, sector byte) ([]byte, error) {
	if track >= md.sectorDisk.Tracks() {
		return nil, fmt.Errorf("ReadLogicalSector expected track between 0 and %d; got %d", md.sectorDisk.Tracks()-1, track)
	}
	if sector >= md.sectorDisk.Sectors() {
		return nil, fmt.Errorf("ReadLogicalSector expected sector between 0 and %d; got %d", md.sectorDisk.Sectors()-1, sector)
	}
	physicalSector := md.logicalToPhysical[int(sector)]
	return md.sectorDisk.ReadPhysicalSector(track, physicalSector)
}

// WriteLogicalSector writes a single logical sector to a disk. It
// expects exactly 256 bytes.
func (md MappedDisk) WriteLogicalSector(track byte, sector byte, data []byte) error {
	if track >= md.sectorDisk.Tracks() {
		return fmt.Errorf("WriteLogicalSector expected track between 0 and %d; got %d", md.sectorDisk.Tracks()-1, track)
	}
	if sector >= md.sectorDisk.Sectors() {
		return fmt.Errorf("WriteLogicalSector expected sector between 0 and %d; got %d", md.sectorDisk.Sectors()-1, sector)
	}
	physicalSector := md.logicalToPhysical[int(sector)]
	return md.sectorDisk.WritePhysicalSector(track, physicalSector, data)
}

// Sectors returns the number of sectors in the disk image.
func (md MappedDisk) Sectors() byte {
	return md.sectorDisk.Sectors()
}

// Tracks returns the number of tracks in the disk image.
func (md MappedDisk) Tracks() byte {
	return md.sectorDisk.Tracks()
}

// Write writes the disk contents to the given file.
func (md MappedDisk) Write(w io.Writer) (n int, err error) {
	return md.sectorDisk.Write(w)
}

// Open opens a disk image by filename.
func Open(filename string) (SectorDisk, error) {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".dsk":
		return LoadDSK(filename)
	}
	return nil, fmt.Errorf("Unimplemented/unknown disk file extension %q", ext)
}
