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

// ProDOSLogicalToPhysicalSectorMap maps logical sector numbers to pysical ones.
// See [UtA2e 9-43 - Sectors vs. Blocks].
var ProDOSLogicalToPhysicalSectorMap = []byte{
	0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0E,
	0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F,
}

// ProDosPhysicalToLogicalSectorMap maps physical sector numbers to logical ones.
// See [UtA2e 9-43 - Sectors vs. Blocks].
var ProDosPhysicalToLogicalSectorMap = []byte{
	0x00, 0x08, 0x01, 0x09, 0x02, 0x0A, 0x03, 0x0B,
	0x04, 0x0C, 0x05, 0x0D, 0x06, 0x0E, 0x07, 0x0F,
}

// TrackSector is a pair of track/sector bytes.
type TrackSector struct {
	Track  byte
	Sector byte
}

// SectorDisk is the interface used to read and write disks by
// physical (matches sector header) sector number.
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
	// Order returns the sector order.
	Order() string
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
	// Order returns the underlying sector ordering.
	Order() string
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

// Order returns the sector order of the underlying sector disk.
func (md MappedDisk) Order() string {
	return md.sectorDisk.Order()
}

// OpenDisk opens a disk image by filename.
func OpenDisk(filename string) (SectorDisk, error) {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".dsk":
		return LoadDSK(filename)
	}
	return nil, fmt.Errorf("Unimplemented/unknown disk file extension %q", ext)
}

// OpenDev opens a device image by filename.
func OpenDev(filename string) (BlockDevice, error) {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".po":
		return LoadDev(filename)
	}
	return nil, fmt.Errorf("Unimplemented/unknown device file extension %q", ext)
}

// Open opens a disk image by filename, returning an Operator.
func Open(filename string) (Operator, error) {
	sd, err := OpenDisk(filename)
	if err == nil {
		var op Operator
		op, err = OperatorForDisk(sd)
		if err == nil {
			return op, nil
		}
	}

	dev, err2 := OpenDev(filename)
	if err2 == nil {
		var op Operator
		op, err2 = OperatorForDevice(dev)
		if err2 != nil {
			return nil, err2
		}
		return op, nil
	}

	return nil, err
}

type DiskBlockDevice struct {
	lsd    LogicalSectorDisk
	blocks uint16
}

// BlockDeviceFromSectorDisk creates a ProDOS block device from a
// SectorDisk. It reads maps ProDOS to physical sectors.
func BlockDeviceFromSectorDisk(sd SectorDisk) (BlockDevice, error) {
	lsd, err := NewMappedDisk(sd, ProDOSLogicalToPhysicalSectorMap)
	if err != nil {
		return nil, err
	}

	return DiskBlockDevice{
		lsd:    lsd,
		blocks: uint16(lsd.Tracks()) / 2 * uint16(lsd.Sectors()),
	}, nil
}

// ReadBlock reads a single block from the device. It always returns
// 512 byes.
func (dbv DiskBlockDevice) ReadBlock(index uint16) (Block, error) {
	var b Block
	if index >= dbv.blocks {
		return b, fmt.Errorf("device has %d blocks; tried to read block %d (index=%d)", dbv.blocks, index+1, index)
	}
	i := int(index) * 2
	sectors := int(dbv.lsd.Sectors())

	track0 := i / sectors
	sector0 := i % sectors
	sector1 := sector0 + 1
	track1 := track0
	if sector1 == sectors {
		sector1 = 0
		track1++
	}

	b0, err := dbv.lsd.ReadLogicalSector(byte(track0), byte(sector0))
	if err != nil {
		return b, fmt.Errorf("error reading first half of block %d (t:%d s:%d): %v", index, track0, sector0, err)
	}
	b1, err := dbv.lsd.ReadLogicalSector(byte(track1), byte(sector1))
	if err != nil {
		return b, fmt.Errorf("error reading second half of block %d (t:%d s:%d): %v", index, track1, sector1, err)
	}
	copy(b[:256], b0)
	copy(b[256:], b1)
	return b, nil
}

// WriteBlock writes a single block to a device. It expects exactly
// 512 bytes.
func (dbv DiskBlockDevice) WriteBlock(index uint16, data Block) error {
	if index >= dbv.blocks {
		return fmt.Errorf("device has %d blocks; tried to read block %d (index=%d)", dbv.blocks, index+1, index)
	}
	i := int(index) * 2
	sectors := int(dbv.lsd.Sectors())

	track0 := i / sectors
	sector0 := i % sectors
	sector1 := sector0 + 1
	track1 := track0
	if sector1 == sectors {
		sector1 = 0
		track1++
	}

	if err := dbv.lsd.WriteLogicalSector(byte(track0), byte(sector0), data[:256]); err != nil {
		return fmt.Errorf("error writing first half of block %d (t:%d s:%d): %v", index, track0, sector0, err)
	}
	if err := dbv.lsd.WriteLogicalSector(byte(track1), byte(sector1), data[256:]); err != nil {
		return fmt.Errorf("error writing second half of block %d (t:%d s:%d): %v", index, track1, sector1, err)
	}

	return nil
}

// Blocks returns the number of blocks on the device.
func (dbv DiskBlockDevice) Blocks() uint16 {
	return dbv.blocks
}

// Order returns the underlying sector or block order of the storage.
func (dbv DiskBlockDevice) Order() string {
	return dbv.lsd.Order()
}

// Write writes the device contents to the given Writer.
func (dbv DiskBlockDevice) Write(w io.Writer) (int, error) {
	return dbv.lsd.Write(w)
}
