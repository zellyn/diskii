// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package dos3 contains routines for working with the on-disk
// structures of Apple DOS 3.
package dos3

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

const (
	// VTOCTrack is the track on a DOS3.3 that holds the VTOC.
	VTOCTrack = 17
	// VTOCSector is the sector on a DOS3.3 that holds the VTOC.
	VTOCSector = 0
)

// DiskSector represents a track and sector.
type DiskSector struct {
	Track  byte
	Sector byte
}

// GetTrack returns the track that a DiskSector was loaded from.
func (ds DiskSector) GetTrack() byte {
	return ds.Track
}

// SetTrack sets the track that a DiskSector was loaded from.
func (ds DiskSector) SetTrack(track byte) {
	ds.Track = track
}

// GetSector returns the sector that a DiskSector was loaded from.
func (ds DiskSector) GetSector() byte {
	return ds.Sector
}

// SetSector sets the sector that a DiskSector was loaded from.
func (ds DiskSector) SetSector(sector byte) {
	ds.Sector = sector
}

// TrackFreeSectors maps the free sectors in a single track.
type TrackFreeSectors [4]byte // Bit map of free sectors in a track

// IsFree returns true if the given sector on a track is free (or if
// sector > 15).
func (t TrackFreeSectors) IsFree(sector byte) bool {
	if sector >= 16 {
		return false
	}
	bits := byte(1) << (sector % 8)
	if sector < 8 {
		return t[1]&bits > 0
	}
	return t[0]&bits > 0
}

// UnusedClear returns true if the unused bytes of the free sector map
// for a track are zeroes (as they're supposed to be).
func (t TrackFreeSectors) UnusedClear() bool {
	return t[2] == 0 && t[3] == 0
}

// DiskFreeSectors maps the free sectors on a disk.
type DiskFreeSectors [50]TrackFreeSectors

// VTOC is the struct used to hold the DOS 3.3 VTOC structure.
// See page 4-2 of Beneath Apple DOS.
type VTOC struct {
	DiskSector
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
	FreeSectors            DiskFreeSectors
}

// Validate checks a VTOC sector to make sure it looks normal.
func (v *VTOC) Validate() error {
	if v.Volume == 255 {
		return fmt.Errorf("expected volume to be 0-254, but got 255")
	}
	if v.DOSRelease != 3 {
		return fmt.Errorf("expected DOS release number to be 3; got %d", v.DOSRelease)
	}
	if v.TrackDirection != 1 && v.TrackDirection != -1 {
		return fmt.Errorf("expected track direction to be 1 or -1; got %d", v.TrackDirection)
	}
	if v.NumTracks != 35 {
		return fmt.Errorf("expected number of tracks to be 35; got %d", v.NumTracks)
	}
	if v.NumSectors != 13 && v.NumSectors != 16 {
		return fmt.Errorf("expected number of sectors per track to be 13 or 16; got %d", v.NumSectors)
	}
	if v.BytesPerSector != 256 {
		return fmt.Errorf("expected 256 bytes per sector; got %d", v.BytesPerSector)
	}
	if v.TrackSectorListMaxSize != 122 {
		return fmt.Errorf("expected 122 track/sector pairs per track/sector list sector; got %d", v.TrackSectorListMaxSize)
	}
	for i, tf := range v.FreeSectors {
		if !tf.UnusedClear() {
			return fmt.Errorf("unused bytes of free-sector list for track %d are not zeroes", i)
		}
	}
	return nil
}

// ToSector marshals the VTOC sector to bytes.
func (v VTOC) ToSector() ([]byte, error) {
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

// FromSector unmarshals the VTOC sector from bytes. Input is
// expected to be exactly 256 bytes.
func (v *VTOC) FromSector(data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("VTOC.FromSector expects exactly 256 bytes; got %d", len(data))
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

// DefaultVTOC returns a new, empty VTOC with values set to their
// defaults.
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
		v.FreeSectors[i] = TrackFreeSectors{}
		if i < 35 {
			v.FreeSectors[i] = TrackFreeSectors([4]byte{0xff, 0xff, 0x00, 0x00})
		}
	}
	return v
}

// CatalogSector is the struct used to hold the DOS 3.3 Catalog
// sector.
type CatalogSector struct {
	DiskSector
	Unused1    byte        // Not used
	NextTrack  byte        // Track number of next catalog sector (usually 11 hex)
	NextSector byte        // Sector number of next catalog sector
	Unused2    [8]byte     // Not used
	FileDescs  [7]FileDesc // File descriptive entries
}

// ToSector marshals the CatalogSector to bytes.
func (cs CatalogSector) ToSector() ([]byte, error) {
	buf := make([]byte, 256)
	buf[0x00] = cs.Unused1
	buf[0x01] = cs.NextTrack
	buf[0x02] = cs.NextSector
	copyBytes(buf[0x03:0x0b], cs.Unused2[:])
	for i, fd := range cs.FileDescs {
		fdBytes := fd.ToBytes()
		copyBytes(buf[0x0b+35*i:0x0b+35*(i+1)], fdBytes)
	}
	return buf, nil
}

// FromSector unmarshals the CatalogSector from bytes. Input is
// expected to be exactly 256 bytes.
func (cs *CatalogSector) FromSector(data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("CatalogSector.FromSector expects exactly 256 bytes; got %d", len(data))
	}

	cs.Unused1 = data[0x00]
	cs.NextTrack = data[0x01]
	cs.NextSector = data[0x02]
	copyBytes(cs.Unused2[:], data[0x03:0x0b])

	for i := range cs.FileDescs {
		cs.FileDescs[i].FromBytes(data[0x0b+35*i : 0x0b+35*(i+1)])
	}
	return nil
}

// Filetype is the type for dos 3.3 filetype+locked status byte.
type Filetype byte

// The DOS3 filetypes.
const (
	// FiletypeLocked is just setting the high bit on other file types.
	FiletypeLocked Filetype = 0x80

	FiletypeText        Filetype = 0x00 // Text file
	FiletypeInteger     Filetype = 0x01 // INTEGER BASIC file
	FiletypeApplesoft   Filetype = 0x02 // APPLESOFT BASIC file
	FiletypeBinary      Filetype = 0x04 // BINARY file
	FiletypeS           Filetype = 0x08 // S type file
	FiletypeRelocatable Filetype = 0x10 // RELOCATABLE object module file
	FiletypeA           Filetype = 0x20 // A type file
	FiletypeB           Filetype = 0x40 // B type file
)

// FileDescStatus is the type used to mark file descriptor status.
type FileDescStatus int

// The three actual file descriptor status values.
const (
	FileDescStatusNormal FileDescStatus = iota
	FileDescStatusDeleted
	FileDescStatusUnused
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

// ToBytes marshals the FileDesc to bytes.
func (fd FileDesc) ToBytes() []byte {
	buf := make([]byte, 35)
	buf[0x00] = fd.TrackSectorListTrack
	buf[0x01] = fd.TrackSectorListSector
	buf[0x02] = byte(fd.Filetype)
	copyBytes(buf[0x03:0x21], fd.Filename[:])
	binary.LittleEndian.PutUint16(buf[0x21:0x23], fd.SectorCount)

	return buf
}

// FromBytes unmarshals the FileDesc from bytes. Input is
// expected to be exactly 35 bytes.
func (fd *FileDesc) FromBytes(data []byte) {
	if len(data) != 35 {
		panic(fmt.Sprintf("FileDesc.FromBytes expects exactly 35 bytes; got %d", len(data)))
	}

	fd.TrackSectorListTrack = data[0x00]
	fd.TrackSectorListSector = data[0x01]
	fd.Filetype = Filetype(data[0x02])
	copyBytes(fd.Filename[:], data[0x03:0x21])
	fd.SectorCount = binary.LittleEndian.Uint16(data[0x21:0x23])
}

// Status returns whether the FileDesc describes a deleted file, a
// normal file, or has never been used.
func (fd *FileDesc) Status() FileDescStatus {
	switch fd.TrackSectorListTrack {
	case 0:
		return FileDescStatusUnused // Never been used.
	case 0xff:
		return FileDescStatusDeleted
	default:
		return FileDescStatusNormal
	}
}

// FilenameString returns the filename of a FileDesc as a normal
// string.
func (fd *FileDesc) FilenameString() string {
	var slice []byte
	if fd.Status() == FileDescStatusDeleted {
		slice = append(slice, fd.Filename[0:len(fd.Filename)-1]...)
	} else {
		slice = append(slice, fd.Filename[:]...)
	}
	for i := range slice {
		slice[i] -= 0x80
	}
	return strings.TrimRight(string(slice), " ")
}

// descriptor returns a types.Descriptor for a FileDesc, but with the
// length set to -1, since we can't know it without reading the file
// contents.
func (fd FileDesc) descriptor() types.Descriptor {
	desc := types.Descriptor{
		Name:    fd.FilenameString(),
		Sectors: int(fd.SectorCount),
		Length:  -1,
		Locked:  (fd.Filetype & FiletypeLocked) > 0,
	}
	switch fd.Filetype & 0x7f {
	case FiletypeText: // Text file
		desc.Type = types.FiletypeASCIIText
	case FiletypeInteger: // INTEGER BASIC file
		desc.Type = types.FiletypeIntegerBASIC
	case FiletypeApplesoft: // APPLESOFT BASIC file
		desc.Type = types.FiletypeApplesoftBASIC
	case FiletypeBinary: // BINARY file
		desc.Type = types.FiletypeBinary
	case FiletypeS: // S type file
		desc.Type = types.FiletypeS
	case FiletypeRelocatable: // RELOCATABLE object module file
		desc.Type = types.FiletypeRelocatable
	case FiletypeA: // A type file
		desc.Type = types.FiletypeNewA
	case FiletypeB: // B type file
		desc.Type = types.FiletypeNewB
	}
	return desc
}

// Contents returns the on-disk contents of a file represented by a
// FileDesc.
func (fd *FileDesc) Contents(diskbytes []byte) ([]byte, error) {
	tsls := []TrackSectorList{}
	nextTrack := fd.TrackSectorListTrack
	nextSector := fd.TrackSectorListSector
	seen := map[disk.TrackSector]bool{}
	for nextTrack != 0 || nextSector != 0 {
		ts := disk.TrackSector{Track: nextTrack, Sector: nextSector}
		if seen[ts] {
			return nil, fmt.Errorf("File %q tries to read TrackSector track=%d sector=%d twice", fd.FilenameString(), nextTrack, nextSector)
		}
		seen[ts] = true
		tsl := TrackSectorList{}
		if err := disk.UnmarshalLogicalSector(diskbytes, &tsl, nextTrack, nextSector); err != nil {
			return nil, err
		}
		tsls = append(tsls, tsl)
		nextTrack = tsl.NextTrack
		nextSector = tsl.NextSector
	}
	data := make([]byte, 0, 256*122*len(tsls))
	for i, tsl := range tsls {
		end := 121
		// If it's the last tsl, stop at the last non-zero TrackSector.
		if i == len(tsls)-1 {
			for j, ts := range tsl.TrackSectors {
				if ts.Track != 0 || ts.Sector != 0 {
					end = j
				}
			}
		}
		for j := 0; j <= end; j++ {
			ts := tsl.TrackSectors[j]
			if ts.Track == 0 && ts.Sector == 0 {
				for k := 0; k < 256; k++ {
					data = append(data, 0)
				}
			} else {
				contents, err := disk.ReadSector(diskbytes, ts.Track, ts.Sector)
				if err != nil {
					return nil, err
				}
				data = append(data, contents...)
			}
		}
	}
	return data, nil
}

// TrackSectorList is the struct used to represent DOS 3.3
// Track/Sector List sectors.
type TrackSectorList struct {
	DiskSector
	Unused1      byte    // Not used
	NextTrack    byte    // Track number of next T/S List sector if one was needed or zero if no more T/S List sectors.
	NextSector   byte    // Sector number of next T/S List sector (if present).
	Unused2      [2]byte // Not used
	SectorOffset uint16  // Sector offset in file of the first sector described by this list.
	Unused3      [5]byte // Not used
	TrackSectors [122]disk.TrackSector
}

// ToSector marshals the TrackSectorList to bytes.
func (tsl TrackSectorList) ToSector() ([]byte, error) {
	buf := make([]byte, 256)
	buf[0x00] = tsl.Unused1
	buf[0x01] = tsl.NextTrack
	buf[0x02] = tsl.NextSector
	copyBytes(buf[0x03:0x05], tsl.Unused2[:])
	binary.LittleEndian.PutUint16(buf[0x05:0x07], tsl.SectorOffset)
	copyBytes(buf[0x07:0x0C], tsl.Unused3[:])

	for i, ts := range tsl.TrackSectors {
		buf[0x0C+i*2] = ts.Track
		buf[0x0D+i*2] = ts.Sector
	}
	return buf, nil
}

// FromSector unmarshals the TrackSectorList from bytes. Input is
// expected to be exactly 256 bytes.
func (tsl *TrackSectorList) FromSector(data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("TrackSectorList.FromSector expects exactly 256 bytes; got %d", len(data))
	}

	tsl.Unused1 = data[0x00]
	tsl.NextTrack = data[0x01]
	tsl.NextSector = data[0x02]
	copyBytes(tsl.Unused2[:], data[0x03:0x05])
	tsl.SectorOffset = binary.LittleEndian.Uint16(data[0x05:0x07])
	copyBytes(tsl.Unused3[:], data[0x07:0x0C])

	for i := range tsl.TrackSectors {
		tsl.TrackSectors[i].Track = data[0x0C+i*2]
		tsl.TrackSectors[i].Sector = data[0x0D+i*2]
	}
	return nil
}

// readCatalogSectors reads the raw CatalogSector structs from a DOS
// 3.3 disk.
func readCatalogSectors(diskbytes []byte, debug bool) ([]CatalogSector, error) {
	v := &VTOC{}
	err := disk.UnmarshalLogicalSector(diskbytes, v, VTOCTrack, VTOCSector)
	if err != nil {
		return nil, err
	}
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid VTOC sector: %v", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "Read VTOC sector: %#v\n", v)
	}

	nextTrack := v.CatalogTrack
	nextSector := v.CatalogSector
	css := []CatalogSector{}
	seen := map[disk.TrackSector]bool{}
	for nextTrack != 0 || nextSector != 0 {
		ts := disk.TrackSector{Track: nextTrack, Sector: nextSector}
		if seen[ts] {
			return nil, fmt.Errorf("Catalog tries to read TrackSector track=%d sector=%d twice", nextTrack, nextSector)
		}
		if nextTrack >= v.NumTracks {
			return nil, fmt.Errorf("catalog sectors can't be in track %d: disk only has %d tracks", nextTrack, v.NumTracks)
		}
		if nextSector >= v.NumSectors {
			return nil, fmt.Errorf("catalog sectors can't be in sector %d: disk only has %d sectors", nextSector, v.NumSectors)
		}
		cs := CatalogSector{}
		err := disk.UnmarshalLogicalSector(diskbytes, &cs, nextTrack, nextSector)
		if err != nil {
			return nil, err
		}
		css = append(css, cs)
		nextTrack = cs.NextTrack
		nextSector = cs.NextSector
	}
	return css, nil
}

// ReadCatalog reads the catalog of a DOS 3.3 disk.
func ReadCatalog(diskbytes []byte, debug bool) (files, deleted []FileDesc, err error) {
	css, err := readCatalogSectors(diskbytes, debug)
	if err != nil {
		return nil, nil, err
	}

	for _, cs := range css {
		for _, fd := range cs.FileDescs {
			switch fd.Status() {
			case FileDescStatusUnused:
				// skip
			case FileDescStatusDeleted:
				deleted = append(deleted, fd)
			case FileDescStatusNormal:
				files = append(files, fd)
			}
		}
	}
	return files, deleted, nil
}

// operator is a types.Operator - an interface for performing
// high-level operations on files and directories.
type operator struct {
	data  []byte
	debug bool
}

var _ types.Operator = operator{}

// operatorName is the keyword name for the operator that undestands
// dos3 disks.
const operatorName = "dos3"

// Name returns the name of the operator.
func (o operator) Name() string {
	return operatorName
}

// HasSubdirs returns true if the underlying operating system on the
// disk allows subdirectories.
func (o operator) HasSubdirs() bool {
	return false
}

// Catalog returns a catalog of disk entries. subdir should be empty
// for operating systems that do not support subdirectories.
func (o operator) Catalog(subdir string) ([]types.Descriptor, error) {
	fds, _, err := ReadCatalog(o.data, o.debug)
	if err != nil {
		return nil, err
	}
	descs := make([]types.Descriptor, 0, len(fds))
	for _, fd := range fds {
		descs = append(descs, fd.descriptor())
	}
	return descs, nil
}

// fileForFilename returns the FileDesc corresponding to the given
// filename, or an error.
func (o operator) fileForFilename(filename string) (FileDesc, error) {
	fds, _, err := ReadCatalog(o.data, o.debug)
	if err != nil {
		return FileDesc{}, err
	}
	for _, fd := range fds {
		if fd.FilenameString() == filename {
			return fd, nil
		}
	}
	return FileDesc{}, fmt.Errorf("Filename %q not found", filename)
}

// GetFile retrieves a file by name.
func (o operator) GetFile(filename string) (types.FileInfo, error) {
	fd, err := o.fileForFilename(filename)
	if err != nil {
		return types.FileInfo{}, err
	}
	desc := fd.descriptor()
	data, err := fd.Contents(o.data)
	if err != nil {
		return types.FileInfo{}, err
	}

	fi := types.FileInfo{
		Descriptor: desc,
		Data:       data,
	}

	errType := "UNKNOWN"
	switch fd.Filetype & 0x7f {
	case FiletypeText: // Text file
		for data[len(data)-1] == 0 {
			data = data[:len(data)-1]
		}
		fi.Descriptor.Length = len(data)
		fi.Data = data
		return fi, nil

	case FiletypeInteger, FiletypeApplesoft, FiletypeBinary:
		switch fd.Filetype & 0x7f {
		case FiletypeApplesoft:
			fi.StartAddress = 0x801
		case FiletypeInteger:
			// TODO(zellyn): figure out what address integer basic programs are stored at.
		case FiletypeBinary:
			fi.StartAddress = uint16(data[0]) + uint16(data[1])<<8
			data = data[2:]
		}
		length := int(data[0]) + int(data[1])*256
		data = data[2 : length+2]
		fi.Descriptor.Length = length
		fi.Data = data
		return fi, nil

	case FiletypeS: // S type file
		errType = "S"
	case FiletypeRelocatable: // RELOCATABLE object module file
		errType = "REL"
	case FiletypeA: // A type file
		errType = "A"
	case FiletypeB: // B type file
		errType = "B"
	}

	return types.FileInfo{}, fmt.Errorf("%s does not yet implement `GetFile` for filetype %s", operatorName, errType)
}

// Delete deletes a file by name. It returns true if the file was
// deleted, false if it didn't exist.
func (o operator) Delete(filename string) (bool, error) {
	return false, fmt.Errorf("%s does not implement Delete yet", operatorName)
}

// PutFile writes a file by name. If the file exists and overwrite
// is false, it returns with an error. Otherwise it returns true if
// an existing file was overwritten.
func (o operator) PutFile(fileInfo types.FileInfo, overwrite bool) (existed bool, err error) {
	return false, fmt.Errorf("%s does not implement PutFile yet", operatorName)
}

// OperatorFactory is a types.OperatorFactory for DOS 3.3 disks.
type OperatorFactory struct {
}

// Name returns the name of the operator.
func (of OperatorFactory) Name() string {
	return operatorName
}

// SeemsToMatch returns true if the []byte disk image seems to match the
// system of this operator.
func (of OperatorFactory) SeemsToMatch(diskbytes []byte, debug bool) bool {
	// For now, just return true if we can run Catalog successfully.
	_, _, err := ReadCatalog(diskbytes, debug)
	if err != nil {
		return false
	}
	return true
}

// Operator returns an Operator for the []byte disk image.
func (of OperatorFactory) Operator(diskbytes []byte, debug bool) (types.Operator, error) {
	return operator{data: diskbytes, debug: debug}, nil
}
