// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// dsk.go contains logic for reading ".dsk" disk images.

package disk

import (
	"fmt"
	"io"
	"io/ioutil"
)

// DSK represents a .dsk disk image.
type DSK struct {
	data             []byte // The actual data in the file
	sectors          byte   // Number of sectors per track
	physicalToStored []byte // Map of physical on-disk sector numbers to sectors in the disk image
	bytesPerTrack    int    // Number of bytes per track
	tracks           byte   // Number of tracks
}

var _ SectorDisk = (*DSK)(nil)

// LoadDSK loads a .dsk image from a file.
func LoadDSK(filename string) (DSK, error) {
	bb, err := ioutil.ReadFile(filename)
	if err != nil {
		return DSK{}, err
	}
	// TODO(zellyn): handle 13-sector disks.
	if len(bb) != DOS33DiskBytes {
		return DSK{}, fmt.Errorf("Expected file %q to contain %d bytes, but got %d.", filename, DOS33DiskBytes, len(bb))
	}
	return DSK{
		data:             bb,
		sectors:          16,
		physicalToStored: Dos33PhysicalToLogicalSectorMap,
		bytesPerTrack:    16 * 256,
		tracks:           DOS33Tracks,
	}, nil
}

// ReadPhysicalSector reads a single physical sector from the disk. It
// always returns 256 byes.
func (d DSK) ReadPhysicalSector(track byte, sector byte) ([]byte, error) {
	if track >= d.tracks {
		return nil, fmt.Errorf("ReadPhysicalSector expected track between 0 and %d; got %d", d.tracks-1, track)
	}
	if sector >= d.sectors {
		return nil, fmt.Errorf("ReadPhysicalSector expected sector between 0 and %d; got %d", d.sectors-1, sector)
	}

	storedSector := d.physicalToStored[int(sector)]
	start := int(track)*d.bytesPerTrack + 256*int(storedSector)
	buf := make([]byte, 256)
	copy(buf, d.data[start:start+256])
	return buf, nil
}

// WritePhysicalSector writes a single physical sector to a disk. It
// expects exactly 256 bytes.
func (d DSK) WritePhysicalSector(track byte, sector byte, data []byte) error {
	if track >= d.tracks {
		return fmt.Errorf("WritePhysicalSector expected track between 0 and %d; got %d", d.tracks-1, track)
	}
	if sector >= d.sectors {
		return fmt.Errorf("WritePhysicalSector expected sector between 0 and %d; got %d", d.sectors-1, sector)
	}
	if len(data) != 256 {
		return fmt.Errorf("WritePhysicalSector expects data of length 256; got %d", len(data))
	}

	storedSector := d.physicalToStored[int(sector)]
	start := int(track)*d.bytesPerTrack + 256*int(storedSector)
	copy(d.data[start:start+256], data)
	return nil
}

// Sectors returns the number of sectors on the DSK image.
func (d DSK) Sectors() byte {
	return d.sectors
}

// Tracks returns the number of tracks on the DSK image.
func (d DSK) Tracks() byte {
	return d.tracks
}

// Write writes the disk contents to the given file.
func (d DSK) Write(w io.Writer) (n int, err error) {
	return w.Write(d.data)
}
