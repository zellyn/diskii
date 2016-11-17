// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package supermon contains routines for working with the on-disk
// structures of NakedOS/Super-Mon disks.
package supermon

// TODO(zellyn): remove panics.

import (
	"fmt"
	"strconv"
	"strings"

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

// SectorMap is the list of sectors by file. It's always 560 bytes
// long (35 tracks * 16 sectors).
type SectorMap []byte

// LoadSectorMap loads a NakedOS sector map.
func LoadSectorMap(sd disk.SectorDisk) (SectorMap, error) {
	sm := SectorMap(make([]byte, 560))
	sector09, err := sd.ReadPhysicalSector(0, 9)
	if err != nil {
		return sm, err
	}
	sector0A, err := sd.ReadPhysicalSector(0, 0xA)
	if err != nil {
		return sm, err
	}
	sector0B, err := sd.ReadPhysicalSector(0, 0xB)
	if err != nil {
		return sm, err
	}
	copy(sm[0:0x30], sector09[0xd0:])
	copy(sm[0x30:0x130], sector0A)
	copy(sm[0x130:0x230], sector0B)
	return sm, nil
}

// Verify checks that we actually have a NakedOS disk.
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

// FileForSector returns the file that owns the given track/sector, or
// zero if the track or sector is too high.
func (sm SectorMap) FileForSector(track, sector byte) byte {
	if track >= 35 {
		return FileIllegal
	}
	if sector >= 16 {
		return FileIllegal
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

// ReadFile reads the contents of a file.
func (sm SectorMap) ReadFile(sd disk.SectorDisk, file byte) ([]byte, error) {
	var result []byte
	for _, ts := range sm.SectorsForFile(file) {
		bytes, err := sd.ReadPhysicalSector(ts.Track, ts.Sector)
		if err != nil {
			return nil, err
		}
		result = append(result, bytes...)
	}
	return result, nil
}

// Symbol represents a single Super-Mon symbol.
type Symbol struct {
	// Address is the memory address the symbol points to, or 0 for an
	// empty symbol table entry.
	Address uint16
	// Name is the name of the symbol.
	Name string
	// Link is the index of the next symbol in the symbol chain for this
	// hash key, or -1 if none.
	Link int
}

// decodeSymbol decodes a Super-Mon encoded symbol table entry,
// returning the string representation.
func decodeSymbol(five []byte, extra byte) string {
	result := ""
	value := uint64(five[0]) + uint64(five[1])<<8 + uint64(five[2])<<16 + uint64(five[3])<<24 + uint64(five[4])<<32 + uint64(extra)<<40
	for value&0x1f > 0 {
		if value&0x1f < 27 {
			result = result + string(value&0x1f+'@')
			value >>= 5
			continue
		}
		if value&0x20 == 0 {
			result = result + string((value&0x1f)-0x1b+'0')
		} else {
			result = result + string((value&0x1f)-0x1b+'5')
		}
		value >>= 6
	}
	return result
}

// SymbolTable represents an entire Super-Mon symbol table. It'll
// always be 819 entries long, because it includes blanks.
type SymbolTable []Symbol

// ReadSymbolTable reads the symbol table from a disk. If there are
// problems with the symbol table (like it doesn't exist, or the link
// pointers are problematic), it'll return nil and an error.
func (sm SectorMap) ReadSymbolTable(sd disk.SectorDisk) (SymbolTable, error) {
	table := make(SymbolTable, 0, 819)
	symtbl1, err := sm.ReadFile(sd, 3)
	if err != nil {
		return nil, err
	}
	if len(symtbl1) != 0x1000 {
		return nil, fmt.Errorf("expected file FSYMTBL1(0x3) to be 0x1000 bytes long; got 0x%04X", len(symtbl1))
	}
	symtbl2, err := sm.ReadFile(sd, 4)
	if err != nil {
		return nil, err
	}
	if len(symtbl2) != 0x1000 {
		return nil, fmt.Errorf("expected file FSYMTBL1(0x4) to be 0x1000 bytes long; got 0x%04X", len(symtbl2))
	}

	five := []byte{0, 0, 0, 0, 0}
	for i := 0; i < 0x0fff; i += 5 {
		address := uint16(symtbl1[i]) + uint16(symtbl1[i+1])<<8
		if address == 0 {
			table = append(table, Symbol{})
			continue
		}
		linkAddr := uint16(symtbl1[i+2]) + uint16(symtbl1[i+3])<<8
		link := -1
		if linkAddr != 0 {
			if linkAddr < 0xD000 || linkAddr >= 0xDFFF {
				return nil, fmt.Errorf("Expected symbol table link address between 0xD000 and 0xDFFE; got 0x%04X", linkAddr)
			}
			if (linkAddr-0xD000)%5 != 0 {
				return nil, fmt.Errorf("Expected symbol table link address to 0xD000+5x; got 0x%04X", linkAddr)
			}
			link = (int(linkAddr) - 0xD000) % 5
		}
		extra := symtbl1[i+4]
		copy(five, symtbl2[i:i+5])
		name := decodeSymbol(five, extra)
		symbol := Symbol{
			Address: address,
			Name:    name,
			Link:    link,
		}
		table = append(table, symbol)
	}

	// TODO(zellyn): check link addresses.

	return table, nil
}

// SymbolsByAddress returns a map of addresses to slices of symbols.
func (st SymbolTable) SymbolsByAddress() map[uint16][]Symbol {
	result := map[uint16][]Symbol{}
	for _, symbol := range st {
		if symbol.Address != 0 {
			result[symbol.Address] = append(result[symbol.Address], symbol)
		}
	}
	return result
}

// NameForFile returns a string representation of a filename:
// either DFxx, or a symbol, if one exists for that value.
func NameForFile(file byte, symbols []Symbol) string {
	if len(symbols) > 0 {
		return symbols[0].Name
	}
	return fmt.Sprintf("DF%02X", file)
}

// FileForName returns a byte file number for a representation of a
// filename: either DFxx, or a symbol, if one exists with the given
// name and points to a DFxx address.
func (st SymbolTable) FileForName(filename string) (byte, error) {
	if addr, err := strconv.ParseUint(filename, 16, 16); err == nil {
		if addr > 0xDF00 && addr < 0xDFFE {
			return byte(addr - 0xDF00), nil
		}
	}

	for _, symbol := range st {
		if strings.EqualFold(symbol.Name, filename) {
			if symbol.Address > 0xDF00 && symbol.Address < 0xDFFE {
				return byte(symbol.Address - 0xDF00), nil
			}
			break
		}
	}

	return 0, fmt.Errorf("invalid filename: %q", filename)
}

// operator is a disk.Operator - an interface for performing
// high-level operations on files and directories.
type operator struct {
	sd      disk.SectorDisk
	sm      SectorMap
	st      SymbolTable
	symbols map[uint16][]Symbol
}

var _ disk.Operator = operator{}

// operatorName is the keyword name for the operator that undestands
// NakedOS/Super-Mon disks.
const operatorName = "nakedos"

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
func (o operator) Catalog(subdir string) ([]disk.Descriptor, error) {
	var descs []disk.Descriptor
	sectorsByFile := o.sm.SectorsByFile()
	for file := byte(1); file < FileReserved; file++ {
		l := len(sectorsByFile[file])
		if l == 0 {
			continue
		}
		fileAddr := 0xDF00 + uint16(file)
		descs = append(descs, disk.Descriptor{
			Name:    NameForFile(file, o.symbols[fileAddr]),
			Sectors: l,
			Length:  l * 256,
			Locked:  false,
		})
	}
	return descs, nil
}

// GetFile retrieves a file by name.
func (o operator) GetFile(filename string) (disk.FileInfo, error) {
	file, err := o.st.FileForName(filename)
	if err != nil {
		return disk.FileInfo{}, err
	}
	data, err := o.sm.ReadFile(o.sd, file)
	if err != nil {
		return disk.FileInfo{}, fmt.Errorf("error reading file DF%02x: %v", file, err)
	}
	if len(data) == 0 {
		return disk.FileInfo{}, fmt.Errorf("file DF%02x not fount", file)
	}
	desc := disk.Descriptor{
		Name:    NameForFile(file, o.symbols[0xDF00+uint16(file)]),
		Sectors: len(data) / 256,
		Length:  len(data),
		Locked:  false,
		Type:    disk.FiletypeBinary,
	}
	return disk.FileInfo{
		Descriptor: desc,
		Data:       data,
	}, nil
}

// operatorFactory is the factory that returns supermon operators
// given disk images.
func operatorFactory(sd disk.SectorDisk) (disk.Operator, error) {
	sm, err := LoadSectorMap(sd)
	if err != nil {
		return nil, err
	}
	if err := sm.Verify(); err != nil {
		return nil, err
	}

	op := operator{sd: sd, sm: sm}

	st, err := sm.ReadSymbolTable(sd)
	if err == nil {
		op.st = st
		op.symbols = st.SymbolsByAddress()
	}

	return op, nil
}

func init() {
	disk.RegisterOperatorFactory(operatorName, operatorFactory)
}
