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
func (sm SectorMap) ReadFile(sd disk.SectorDisk, file byte) []byte {
	var result []byte
	for _, ts := range sm.SectorsForFile(file) {
		bytes, err := sd.ReadPhysicalSector(ts.Track, ts.Sector)
		if err != nil {
			panic(err.Error())
		}
		result = append(result, bytes...)
	}
	return result
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
	symtbl1 := sm.ReadFile(sd, 3)
	if len(symtbl1) != 0x1000 {
		return nil, fmt.Errorf("expected file FSYMTBL1(0x3) to be 0x1000 bytes long; got 0x%04X", len(symtbl1))
	}
	symtbl2 := sm.ReadFile(sd, 4)
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
