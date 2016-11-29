// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package supermon contains routines for working with the on-disk
// structures of NakedOS/Super-Mon disks.
package supermon

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/zellyn/diskii/lib/disk"
	"github.com/zellyn/diskii/lib/errors"
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

// Persist writes the current contenst of a sector map back back to
// disk.
func (sm SectorMap) Persist(sd disk.SectorDisk) error {
	sector09, err := sd.ReadPhysicalSector(0, 9)
	if err != nil {
		return err
	}
	copy(sector09[0xd0:], sm[0:0x30])
	if err := sd.WritePhysicalSector(0, 9, sector09); err != nil {
		return err
	}
	if err := sd.WritePhysicalSector(0, 0xA, sm[0x30:0x130]); err != nil {
		return err
	}
	if err := sd.WritePhysicalSector(0, 0xB, sm[0x130:0x230]); err != nil {
		return err
	}
	return nil
}

// FreeSectors returns the number of blocks free in a sector map.
func (sm SectorMap) FreeSectors() int {
	count := 0
	for _, file := range sm {
		if file == FileFree {
			count++
		}
	}
	return count
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

// SetFileForSector sets the file that owns the given track/sector, or
// returns an error if the track or sector is too high.
func (sm SectorMap) SetFileForSector(track, sector, file byte) error {
	if track >= 35 {
		return fmt.Errorf("track %d >34", track)
	}
	if sector >= 16 {
		return fmt.Errorf("sector %d >15", sector)
	}
	if file == FileIllegal || file == FileFree || file == FileReserved {
		return fmt.Errorf("illegal file number: 0x%0X", file)
	}
	sm[int(track)*16+int(sector)] = file
	return nil
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

// Delete deletes a file from the sector map. It does not persist the changes.
func (sm SectorMap) Delete(file byte) {
	for i, f := range sm {
		if f == file {
			sm[i] = FileFree
		}
	}
}

// WriteFile writes the contents of a file.
func (sm SectorMap) WriteFile(sd disk.SectorDisk, file byte, contents []byte, overwrite bool) error {
	sectorsNeeded := (len(contents) + 255) / 256
	cts := make([]byte, 256*sectorsNeeded)
	copy(cts, contents)
	existing := len(sm.SectorsForFile(file))
	free := sm.FreeSectors() + existing
	if free < sectorsNeeded {
		return errors.OutOfSpacef("file %d requires %d sectors, but only %d are available", file, sectorsNeeded, free)
	}
	if existing > 0 {
		if !overwrite {
			return errors.FileExistsf("file %d already exists", file)
		}
		sm.Delete(file)
	}

	i := 0
OUTER:
	for track := byte(0); track < sd.Tracks(); track++ {
		for sector := byte(0); sector < sd.Sectors(); sector++ {
			if sm.FileForSector(track, sector) == FileFree {
				if err := sd.WritePhysicalSector(track, sector, cts[i*256:(i+1)*256]); err != nil {
					return err
				}
				if err := sm.SetFileForSector(track, sector, file); err != nil {
					return err
				}
				i++
				if i == sectorsNeeded {
					break OUTER
				}
			}
		}
	}
	if err := sm.Persist(sd); err != nil {
		return err
	}
	return nil
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

// encodeSymbol encodes a symbol name into the five+1 bytes used in a
// Super-Mon encoded symbol table entry. The returned byte array will
// always be six bytes long. If it can't be encoded, it returns an
// error. Empty strings are encoded as all zeros.
func encodeSymbol(name string) (six []byte, err error) {
	if name == "" {
		six := make([]byte, 6)
		return six, nil
	}
	if len(name) > 9 {
		return nil, fmt.Errorf("invalid Super-Mon symbol %q: too long", name)
	}
	if len(name) < 3 {
		return nil, fmt.Errorf("invalid Super-Mon symbol %q: too short", name)
	}
	nm := []byte(strings.ToUpper(name))
	value := uint64(0)
	bits := 0
	for i := len(nm) - 1; i >= 0; i-- {
		ch := nm[i]
		switch {
		case 'A' <= ch && ch <= 'Z':
			value = value<<5 + uint64(ch-'@')
			bits += 5
		case '0' <= ch && ch <= '4':
			value = value<<6 + 0x1b + uint64(ch-'0')
			bits += 6
		case '5' <= ch && ch <= '9':
			value = value<<6 + 0x3b + uint64(ch-'5')
			bits += 6
		}
		if bits > 48 {
			return nil, fmt.Errorf("invalid Super-Mon symbol %q: too long", name)
		}
	}
	eight := make([]byte, 8)
	six = make([]byte, 6)
	binary.LittleEndian.PutUint64(eight, value)
	copy(six, eight)
	return six, nil
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
			link = (int(linkAddr) - 0xD000) / 5
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

	for i, sym := range table {
		if sym.Address != 0 && sym.Link != -1 {
			if sym.Link == i {
				return nil, fmt.Errorf("Symbol %q (0x%04X) links to itself", sym.Name, sym.Address)
			}
			linkSym := table[sym.Link]
			if addrHash(sym.Address) != addrHash(linkSym.Address) {
				return nil, fmt.Errorf("Symbol %q (0x%04X) with hash 0x%02X links to symbol %q (0x%04X) with hash 0x%02X",
					sym.Name, sym.Address, addrHash(sym.Address), linkSym.Name, linkSym.Address, addrHash(linkSym.Address))
			}
		}
	}

	return table, nil
}

// WriteSymbolTable writes a symbol table to a disk.
func (sm SectorMap) WriteSymbolTable(sd disk.SectorDisk, st SymbolTable) error {
	symtbl1 := make([]byte, 0x1000)
	symtbl2 := make([]byte, 0x1000)
	for i, sym := range st {
		offset := i * 5
		linkAddr := 0
		six, err := encodeSymbol(sym.Name)
		if err != nil {
			return err
		}
		if sym.Link != -1 {
			linkAddr = sym.Link*5 + 0xD000
		}
		symtbl1[offset] = byte(sym.Address % 256)
		symtbl1[offset+1] = byte(sym.Address >> 8)
		symtbl1[offset+2] = byte(linkAddr % 256)
		symtbl1[offset+3] = byte(linkAddr >> 8)
		symtbl1[offset+4] = six[5]
		copy(symtbl2[offset:offset+5], six)
	}
	if err := sm.WriteFile(sd, 3, symtbl1, true); err != nil {
		return fmt.Errorf("unable to write first half of symbol table: %v", err)
	}
	if err := sm.WriteFile(sd, 4, symtbl2, true); err != nil {
		return fmt.Errorf("unable to write first second of symbol table: %v", err)
	}
	return nil
}

// addrHash computes the SuperMon hash for an address.
func addrHash(addr uint16) byte {
	return (byte(addr) ^ byte(addr>>8)) & 0x7f
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

// DeleteSymbol deletes an existing symbol. Returns true if the named
// symbol was found.
func (st SymbolTable) DeleteSymbol(name string) bool {
	for i, sym := range st {
		if strings.EqualFold(name, sym.Name) {
			sym.Name = ""
			sym.Address = 0
			for j := range st {
				if i == j {
					continue
				}
				if st[j].Link == i {
					st[j].Link = sym.Link
					break
				}
			}
			st[i] = sym
			return true
		}
	}
	return false
}

// AddSymbol adds a new symbol. If a symbol with the given name
// already exists with a different address, it deletes it first.
func (st SymbolTable) AddSymbol(name string, address uint16) error {
	if address == 0 {
		return fmt.Errorf("cannot set symbol %q to address 0")
	}
	hash := addrHash(address)
	pos := -1
	for j, sym := range st {
		if strings.EqualFold(name, sym.Name) {
			// If we can, simply update the address.
			if addrHash(sym.Address) == hash {
				st[j].Address = address
				return nil
			}
			st.DeleteSymbol(name)
			pos = j
			break
		}
		if pos == -1 && sym.Address == 0 {
			pos = j
		}
	}
	for j, sym := range st {
		if addrHash(sym.Address) == hash && sym.Link == -1 {
			st[j].Link = pos
			break
		}
	}
	st[pos].Name = name
	st[pos].Address = address
	st[pos].Link = -1
	return nil
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

	return 0, errors.FileNotFoundf("filename %q not found", filename)
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
			Type:    disk.FiletypeBinary,
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
	fi := disk.FileInfo{
		Descriptor: desc,
		Data:       data,
	}
	if file == 1 {
		fi.StartAddress = 0x1800
	}
	return fi, nil
}

// Delete deletes a file by name. It returns true if the file was
// deleted, false if it didn't exist.
func (o operator) Delete(filename string) (bool, error) {
	file, err := o.st.FileForName(filename)
	if err != nil {
		return false, err
	}
	existed := len(o.sm.SectorsForFile(file)) > 0
	o.sm.Delete(file)
	if err := o.sm.Persist(o.sd); err != nil {
		return existed, err
	}
	if o.st != nil {
		changed := o.st.DeleteSymbol(filename)
		if changed {
			if err := o.sm.WriteSymbolTable(o.sd, o.st); err != nil {
				return existed, err
			}
		}
	}
	return existed, nil
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
