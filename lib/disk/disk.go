// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package disk contains routines for reading and writing various disk
// file formats.
package disk

import (
	"fmt"
	"io/ioutil"
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

type SectorDisk interface {
	// ReadLogicalSector reads a single logical sector from the disk. It
	// always returns 256 byes.
	ReadLogicalSector(track byte, sector byte) ([]byte, error)
	// WriteLogicalSector writes a single logical sector to a disk. It
	// expects exactly 256 bytes.
	WriteLogicalSector(track byte, sector byte, data []byte) error
}

const (
	DOS33Tracks  = 35 // Tracks per disk
	DOS33Sectors = 16 // Sectors per track
	// DOS33DiskBytes is the number of bytes on a DOS 3.3 disk.
	DOS33DiskBytes  = 143360             // 35 tracks * 16 sectors * 256 bytes
	DOS33TrackBytes = 256 * DOS33Sectors // Bytes per track
)

// DSK represents a .dsk disk image.
type DSK struct {
	data [DOS33DiskBytes]byte
}

var _ SectorDisk = (*DSK)(nil)

// LoadDSK loads a .dsk image from a file.
func LoadDSK(filename string) (DSK, error) {
	d := DSK{}
	bb, err := ioutil.ReadFile(filename)
	if err != nil {
		return d, err
	}
	if len(bb) != DOS33DiskBytes {
		return d, fmt.Errorf("Expected file %q to contain %d bytes, but got %d.", filename, DOS33DiskBytes, len(bb))
	}
	copy(d.data[:], bb)
	return d, nil
}

// ReadLogicalSector reads a single logical sector from the disk. It
// always returns 256 byes.
func (d DSK) ReadLogicalSector(track byte, sector byte) ([]byte, error) {
	if track >= DOS33Tracks {
		return nil, fmt.Errorf("Expected track between 0 and %d; got %d", DOS33Tracks-1, track)
	}
	if sector >= DOS33Sectors {
		return nil, fmt.Errorf("Expected sector between 0 and %d; got %d", DOS33Sectors-1, sector)
	}

	start := int(track)*DOS33TrackBytes + 256*int(sector)
	buf := make([]byte, 256)
	copy(buf, d.data[start:start+256])
	return buf, nil
}

// WriteLogicalSector writes a single logical sector to a disk. It
// expects exactly 256 bytes.
func (d DSK) WriteLogicalSector(track byte, sector byte, data []byte) error {
	if track >= DOS33Tracks {
		return fmt.Errorf("Expected track between 0 and %d; got %d", DOS33Tracks-1, track)
	}
	if sector >= DOS33Sectors {
		return fmt.Errorf("Expected sector between 0 and %d; got %d", DOS33Sectors-1, sector)
	}
	if len(data) != 256 {
		return fmt.Errorf("WriteLogicalSector expects data of length 256; got %d", len(data))
	}

	start := int(track)*DOS33TrackBytes + 256*int(sector)
	copy(d.data[start:start+256], data)
	return nil
}
