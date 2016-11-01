// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package dos33 contains routines for working with the on-disk structures of DOS 3.3.
package dos33

import (
	"encoding/binary"
	"fmt"
)

type FreeSectorMap [4]byte // Bit map of free sectors in a track

// VTOC is the struct used to hold the DOS 3.3 VTOC structure.
// See page 4-2 of Beneath Apple DOS.
type VTOC struct {
	Unused1       byte     // Not used
	CatalogTrack  byte     // Track number of first catalog sector
	CatalogSector byte     // Sector number of first catalog sector
	DOSRelease    byte     // Release number of DOS used to INIT this diskette
	Unused2       [2]byte  // Not used
	Volume        byte     // Diskette volume number (1-254)
	Unused3       [32]byte // Not used
	// Maximum number of track/secotr pairs which will fit in one file
	// track/sector list sector (122 for 256 byte sectors)
	TrackSectorListMaxSize byte
	Unused4                [8]byte // Not used
	LastTrack              byte    // Last track where sectors were allocated
	TrackDirection         int8    // Direction of track allocation (+1 or -1)
	Unused5                [2]byte
	NumTracks              byte   // Number of tracks per diskette (normally 35)
	NumSectors             byte   // Number of sectors per track (13 or 16)
	BytesPerSector         uint16 // Number of bytes per sector (LO/HI format)
	FreeSectors            [50]FreeSectorMap
}

// MarshalBinary marshals the VTOC sector to bytes. Error is always nil.
func (v VTOC) MarshalBinary() (data []byte, err error) {
	buf := make([]byte, 256)
	buf[0x00] = v.Unused1
	buf[0x01] = v.CatalogTrack
	buf[0x02] = v.CatalogSector
	buf[0x03] = v.DOSRelease
	copyBytes(buf[0x04:0x06], v.Unused2[:])
	buf[0x06] = v.Volume
	copyBytes(buf[0x07:0x27], v.Unused3[:])
	buf[0x27] = v.TrackSectorListMaxSize
	copyBytes(buf[0x28:0x30], v.Unused4[:])
	buf[0x30] = v.LastTrack
	buf[0x31] = byte(v.TrackDirection)
	copyBytes(buf[0x32:0x34], v.Unused5[:])
	buf[0x34] = v.NumTracks
	buf[0x35] = v.NumSectors
	binary.LittleEndian.PutUint16(buf[0x36:0x38], v.BytesPerSector)
	for i, m := range v.FreeSectors {
		copyBytes(buf[0x38+4*i:0x38+4*i+4], m[:])
	}
	return buf, nil
}

// copyBytes is just like the builtin copy, but just for byte slices,
// and it checks that dst and src have the same length.
func copyBytes(dst, src []byte) int {
	if len(dst) != len(src) {
		panic(fmt.Sprintf("copyBytes called with differing lengths %d and %d", len(dst), len(src)))
	}
	return copy(dst, src)
}

// UnmarshalBinary unmarshals the VTOC sector from bytes. Input is
// expected to be exactly 256 bytes.
func (v *VTOC) UnmarshalBinary(data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("VTOC.UnmarshalBinary expects exactly 256 bytes; got %d", len(data))
	}

	v.Unused1 = data[0x00]
	v.CatalogTrack = data[0x01]
	v.CatalogSector = data[0x02]
	v.DOSRelease = data[0x03]
	copyBytes(v.Unused2[:], data[0x04:0x06])
	v.Volume = data[0x06]
	copyBytes(v.Unused3[:], data[0x07:0x27])
	v.TrackSectorListMaxSize = data[0x27]
	copyBytes(v.Unused4[:], data[0x28:0x30])
	v.LastTrack = data[0x30]
	v.TrackDirection = int8(data[0x31])
	copyBytes(v.Unused5[:], data[0x32:0x34])
	v.NumTracks = data[0x34]
	v.NumSectors = data[0x35]
	v.BytesPerSector = binary.LittleEndian.Uint16(data[0x36:0x38])
	for i := range v.FreeSectors {
		copyBytes(v.FreeSectors[i][:], data[0x38+4*i:0x38+4*i+4])
	}

	return nil
}

func DefaultVTOC() VTOC {
	v := VTOC{
		CatalogTrack:           0x11,
		CatalogSector:          0x0f,
		DOSRelease:             0x03,
		Volume:                 0x01,
		TrackSectorListMaxSize: 122,
		LastTrack:              0x00, // TODO(zellyn): what should this be?
		TrackDirection:         1,
		NumTracks:              0x23,
		NumSectors:             0x10,
		BytesPerSector:         0x100,
	}
	for i := range v.FreeSectors {
		v.FreeSectors[i] = FreeSectorMap{}
		if i < 35 {
			v.FreeSectors[i] = FreeSectorMap([4]byte{0xff, 0xff, 0x00, 0x00})
		}
	}
	return v
}

// CatalogSector is the struct used to hold the DOS 3.3 Catalog
// sector.
type CatalogSector struct {
	Unused1    byte        // Not used
	NextTrack  byte        // Track number of next catalog sector (usually 11 hex)
	NextSector byte        // Sector number of next catalog sector
	Unused2    [8]byte     // Not used
	FileDescs  [7]FileDesc // File descriptive entries
}

// MarshalBinary marshals the CatalogSector to bytes. Error is always nil.
func (cs CatalogSector) MarshalBinary() (data []byte, err error) {
	buf := make([]byte, 256)
	buf[0x00] = cs.Unused1
	buf[0x01] = cs.NextTrack
	buf[0x02] = cs.NextSector
	copyBytes(buf[0x03:0x0b], cs.Unused2[:])
	for i, fd := range cs.FileDescs {
		fdBytes, _ := fd.MarshalBinary()
		copyBytes(buf[0x0b+35*i:0x0b+35*(i+1)], fdBytes)
	}
	return buf, nil
}

// UnmarshalBinary unmarshals the CatalogSector from bytes. Input is
// expected to be exactly 256 bytes.
func (cs *CatalogSector) UnmarshalBinary(data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("CatalogSector.UnmarshalBinary expects exactly 256 bytes; got %d", len(data))
	}

	cs.Unused1 = data[0x00]
	cs.NextTrack = data[0x01]
	cs.NextSector = data[0x02]
	copyBytes(cs.Unused2[:], data[0x03:0x0b])

	for i := range cs.FileDescs {
		if err := cs.FileDescs[i].UnmarshalBinary(data[0x0b+35*i : 0x0b+35*(i+1)]); err != nil {
			return err
		}
	}

	return nil
}

type Filetype byte

const (
	// Hex 80+file type - file is locked
	// Hex 00+file type - file is not locked
	FiletypeLocked Filetype = 0x80

	FileTypeText        Filetype = 0x00 // Text file
	FileTypeInteger     Filetype = 0x01 // INTEGER BASIC file
	FileTypeApplesoft   Filetype = 0x02 // APPLESOFT BASIC file
	FileTypeBinary      Filetype = 0x04 // BINARY file
	FileTypeS           Filetype = 0x08 // S type file
	FileTypeRelocatable Filetype = 0x10 // RELOCATABLE object module file
	FileTypeA           Filetype = 0x20 // A type file
	FileTypeB           Filetype = 0x40 // B type file
)

// FileDesc is the struct used to represent the DOS 3.3 File
// Descriptive entry.
type FileDesc struct {
	// Track of first track/sector list sector. If this is a deleted
	// file, this byte contains a hex FF and the original track number
	// is copied to the last byte of the file name field (BYTE 20). If
	// this byte contains a hex 00, the entry is assumed to never have
	// been used and is available for use. (This means track 0 can never
	// be used for data even if the DOS image is "wiped" from the
	// diskette.)
	TrackSectorListTrack  byte
	TrackSectorListSector byte     // Sector of first track/sector list sector
	Filetype              Filetype // File type and flags
	Filename              [30]byte // File name (30 characters) Length of file in
	// sectors (LO/HI format). The CATALOG command will only format the
	// LO byte of this length giving 1-255 but a full 65,535 may be
	// stored here.
	SectorCount uint16
}

// MarshalBinary marshals the FileDesc to bytes. Error is always nil.
func (fd FileDesc) MarshalBinary() (data []byte, err error) {
	buf := make([]byte, 35)
	buf[0x00] = fd.TrackSectorListTrack
	buf[0x01] = fd.TrackSectorListSector
	buf[0x02] = byte(fd.Filetype)
	copyBytes(buf[0x03:0x21], fd.Filename[:])
	binary.LittleEndian.PutUint16(buf[0x21:0x23], fd.SectorCount)

	return buf, nil
}

// UnmarshalBinary unmarshals the FileDesc from bytes. Input is
// expected to be exactly 35 bytes.
func (fd *FileDesc) UnmarshalBinary(data []byte) error {
	if len(data) != 35 {
		return fmt.Errorf("FileDesc.UnmarshalBinary expects exactly 35 bytes; got %d", len(data))
	}

	fd.TrackSectorListTrack = data[0x00]
	fd.TrackSectorListSector = data[0x01]
	fd.Filetype = Filetype(data[0x02])
	copyBytes(data[0x03:0x21], fd.Filename[:])
	fd.SectorCount = binary.LittleEndian.Uint16(data[0x21:0x23])

	return nil
}
