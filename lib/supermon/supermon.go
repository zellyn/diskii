// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package supermon contains routines for working with the on-disk
// structures of NakedOS/Super-Mon disks.
package supermon

import (
	"encoding/binary"
	"fmt"
	"io"
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

// FirstFreeFile returns the first file number that isn't already
// used. It returns 0 if all are already used.
func (sm SectorMap) FirstFreeFile() byte {
	for file := byte(0x01); file < 0xfe; file++ {
		sectors := sm.SectorsForFile(file)
		if len(sectors) == 0 {
			return file
		}
	}
	return 0
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

// WriteFile writes the contents of a file. It returns true if the
// file already existed.
func (sm SectorMap) WriteFile(sd disk.SectorDisk, file byte, contents []byte, overwrite bool) (bool, error) {
	sectorsNeeded := (len(contents) + 255) / 256
	cts := make([]byte, 256*sectorsNeeded)
	copy(cts, contents)
	existing := len(sm.SectorsForFile(file))
	existed := existing > 0
	free := sm.FreeSectors() + existing
	if free < sectorsNeeded {
		return existed, errors.OutOfSpacef("file %d requires %d sectors, but only %d are available", file, sectorsNeeded, free)
	}
	if existed {
		if !overwrite {
			return existed, errors.FileExistsf("file %d already exists", file)
		}
		sm.Delete(file)
	}

	i := 0
OUTER:
	for track := byte(0); track < sd.Tracks(); track++ {
		for sector := byte(0); sector < sd.Sectors(); sector++ {
			if sm.FileForSector(track, sector) == FileFree {
				if err := sd.WritePhysicalSector(track, sector, cts[i*256:(i+1)*256]); err != nil {
					return existed, err
				}
				if err := sm.SetFileForSector(track, sector, file); err != nil {
					return existed, err
				}
				i++
				if i == sectorsNeeded {
					break OUTER
				}
			}
		}
	}
	if err := sm.Persist(sd); err != nil {
		return existed, err
	}
	return existed, nil
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

func (s Symbol) String() string {
	return fmt.Sprintf("{%s:%04X:%d}", s.Name, s.Address, s.Link)
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
	if _, err := sm.WriteFile(sd, 3, symtbl1, true); err != nil {
		return fmt.Errorf("unable to write first half of symbol table: %v", err)
	}
	if _, err := sm.WriteFile(sd, 4, symtbl2, true); err != nil {
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

// SymbolsForAddress returns a slice of symbols for a given address.
func (st SymbolTable) SymbolsForAddress(address uint16) []Symbol {
	result := []Symbol{}
	for _, symbol := range st {
		if symbol.Address == address {
			result = append(result, symbol)
		}
	}
	return result
}

// ByName returns the address of the named symbol, or 0 if it's not in
// the symbol table.
func (st SymbolTable) ByName(name string) uint16 {
	for _, symbol := range st {
		if strings.EqualFold(name, symbol.Name) {
			return symbol.Address
		}
	}
	return 0
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
		return fmt.Errorf("cannot set symbol %q to address 0", name)
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
	if pos == -1 {
		return fmt.Errorf("symbol table full")
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
func NameForFile(file byte, st SymbolTable) string {
	symbols := st.SymbolsForAddress(0xDF00 + uint16(file))
	if len(symbols) > 0 {
		return symbols[0].Name
	}
	return fmt.Sprintf("DF%02X", file)
}

// FullnameForFile returns a string representation of a filename:
// either DFxx, or a DFxx:symbol, if one exists for that value.
func FullnameForFile(file byte, st SymbolTable) string {
	symbols := st.SymbolsForAddress(0xDF00 + uint16(file))
	if len(symbols) > 0 {
		return fmt.Sprintf("DF%02X:%s", file, symbols[0].Name)
	}
	return fmt.Sprintf("DF%02X", file)
}

// parseAddressFilename parses filenames of the form DFxx and returns
// the xx part. Invalid filenames result in 0.
func parseAddressFilename(filename string) byte {
	if addr, err := strconv.ParseUint(filename, 16, 16); err == nil {
		if addr > 0xDF00 && addr < 0xDFFE {
			return byte(addr - 0xDF00)
		}
		if addr > 0x00 && addr < 0xFE {
			return byte(addr)
		}
	}
	return 0
}

// FileForName returns a byte file number for a representation of a
// filename: either DFxx, or a symbol, if one exists with the given
// name and points to a DFxx address.
func (st SymbolTable) FileForName(filename string) (byte, error) {
	if addr := parseAddressFilename(filename); addr != 0 {
		return addr, nil
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

// ParseCompoundSymbol parses an address, symbol, or compound of both
// in the forms XXXX, symbolname, or XXXX:symbolname.
func (st SymbolTable) ParseCompoundSymbol(name string) (address uint16, symAddress uint16, symbol string, err error) {
	if name == "" {
		return 0, 0, "", fmt.Errorf("expected symbol name, got %q", name)
	}
	parts := strings.Split(name, ":")
	if len(parts) > 2 {
		return 0, 0, "", fmt.Errorf("more than one colon in compound address:symbol: %q", name)
	}
	if len(parts) == 1 {
		// If there's a symbol by that name, use it.
		if addr := st.ByName(name); addr != 0 {
			return 0, addr, name, nil
		}
		// If we can parse it as an address, do so.
		if addr, err := strconv.ParseUint(name, 16, 16); err == nil {
			return uint16(addr), 0, "", nil
		}
		// If it's a valid symbol name, assume that's what it is.
		if _, err := encodeSymbol(name); err != nil {
			return 0, 0, name, nil
		}
		return 0, 0, "", fmt.Errorf("%q is not a valid symbol name or address", name)
	}

	if parts[0] == "" {
		return 0, 0, "", fmt.Errorf("empty address part of compound address:symbol: %q", name)
	}
	if parts[1] == "" {
		return 0, 0, "", fmt.Errorf("empty symbol part of compound address:symbol: %q", name)
	}

	// If we can parse it as an address, do so.
	addr, err := strconv.ParseUint(parts[0], 16, 16)
	if err != nil {
		return 0, 0, "", fmt.Errorf("error parsing address part of %q: %v", name, err)
	}
	if _, err := encodeSymbol(parts[1]); err != nil {
		return 0, 0, name, err
	}
	return uint16(addr), st.ByName(parts[1]), parts[1], nil
}

// FilesForCompoundName parses a complex filename of the form DFxx,
// FILENAME, or DFxx:FILENAME, returning the file number before the
// colon, and the file name number after the colon, and the symbol
// name.
func (st SymbolTable) FilesForCompoundName(filename string) (numFile byte, namedFile byte, symbol string, err error) {
	parts := strings.Split(filename, ":")
	if len(parts) > 2 {
		return 0, 0, "", fmt.Errorf("more than one colon in compound filename: %q", filename)
	}
	if len(parts) == 1 {
		numFile = parseAddressFilename(filename)
		if numFile != 0 {
			return numFile, 0, "", nil
		}
		file, err := st.FileForName(filename)
		if err != nil {
			return 0, 0, filename, nil
		}
		return file, file, filename, nil
	}
	numFile = parseAddressFilename(parts[0])
	if numFile == 0 {
		return 0, 0, "", fmt.Errorf("invalid file number: %q", parts[0])
	}
	if numFile2 := parseAddressFilename(parts[1]); numFile2 != 0 {
		return 0, 0, "", fmt.Errorf("cannot use valid file number (%q) as a filename", parts[1])
	}
	namedFile, err = st.FileForName(parts[1])
	if err != nil {
		return numFile, 0, parts[1], nil
	}
	return numFile, namedFile, parts[1], nil
}

// Operator is a disk.Operator - an interface for performing
// high-level operations on files and directories.
type Operator struct {
	SD disk.SectorDisk
	SM SectorMap
	ST SymbolTable
}

var _ disk.Operator = Operator{}

// operatorName is the keyword name for the operator that undestands
// NakedOS/Super-Mon disks.
const operatorName = "nakedos"

// Name returns the name of the Operator.
func (o Operator) Name() string {
	return operatorName
}

// Order returns the sector or block order of the Operator.
func (o Operator) Order() string {
	return o.SD.Order()
}

// HasSubdirs returns true if the underlying operating system on the
// disk allows subdirectories.
func (o Operator) HasSubdirs() bool {
	return false
}

// Catalog returns a catalog of disk entries. subdir should be empty
// for operating systems that do not support subdirectories.
func (o Operator) Catalog(subdir string) ([]disk.Descriptor, error) {
	var descs []disk.Descriptor
	sectorsByFile := o.SM.SectorsByFile()
	for file := byte(1); file < FileReserved; file++ {
		l := len(sectorsByFile[file])
		if l == 0 {
			continue
		}
		descs = append(descs, disk.Descriptor{
			Name:     NameForFile(file, o.ST),
			Fullname: FullnameForFile(file, o.ST),
			Sectors:  l,
			Length:   l * 256,
			Locked:   false,
			Type:     disk.FiletypeBinary,
		})
	}
	return descs, nil
}

// GetFile retrieves a file by name.
func (o Operator) GetFile(filename string) (disk.FileInfo, error) {
	file, err := o.ST.FileForName(filename)
	if err != nil {
		return disk.FileInfo{}, err
	}
	data, err := o.SM.ReadFile(o.SD, file)
	if err != nil {
		return disk.FileInfo{}, fmt.Errorf("error reading file DF%02x: %v", file, err)
	}
	if len(data) == 0 {
		return disk.FileInfo{}, fmt.Errorf("file DF%02x not fount", file)
	}
	desc := disk.Descriptor{
		Name:    NameForFile(file, o.ST),
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
func (o Operator) Delete(filename string) (bool, error) {
	file, err := o.ST.FileForName(filename)
	if err != nil {
		return false, err
	}
	existed := len(o.SM.SectorsForFile(file)) > 0
	o.SM.Delete(file)
	if err := o.SM.Persist(o.SD); err != nil {
		return existed, err
	}
	if o.ST != nil {
		changed := o.ST.DeleteSymbol(filename)
		if changed {
			if err := o.SM.WriteSymbolTable(o.SD, o.ST); err != nil {
				return existed, err
			}
		}
	}
	return existed, nil
}

// PutFile writes a file by name. If the file exists and overwrite
// is false, it returns with an error. Otherwise it returns true if
// an existing file was overwritten.
func (o Operator) PutFile(fileInfo disk.FileInfo, overwrite bool) (existed bool, err error) {
	if fileInfo.Descriptor.Type != disk.FiletypeBinary {
		return false, fmt.Errorf("%s: only binary file type supported", operatorName)
	}
	if fileInfo.Descriptor.Length != len(fileInfo.Data) {
		return false, fmt.Errorf("mismatch between FileInfo.Descriptor.Length (%d) and actual length of FileInfo.Data field (%d)", fileInfo.Descriptor.Length, len(fileInfo.Data))
	}

	numFile, namedFile, symbol, err := o.ST.FilesForCompoundName(fileInfo.Descriptor.Name)
	if err != nil {
		return false, err
	}
	if symbol != "" {
		if o.ST == nil {
			return false, fmt.Errorf("cannot use symbolic names on disks without valid symbol tables in files 0x03 and 0x04")
		}
		if _, err := encodeSymbol(symbol); err != nil {
			return false, err
		}
	}
	if numFile == 0 {
		numFile = o.SM.FirstFreeFile()
		if numFile == 0 {
			return false, fmt.Errorf("all files already used")
		}
	}
	existed, err = o.SM.WriteFile(o.SD, numFile, fileInfo.Data, overwrite)
	if err != nil {
		return existed, err
	}
	if namedFile != numFile && symbol != "" {
		if err := o.ST.AddSymbol(symbol, 0xDF00+uint16(numFile)); err != nil {
			return existed, err
		}
		if err := o.SM.WriteSymbolTable(o.SD, o.ST); err != nil {
			return existed, err
		}
	}
	return existed, nil
}

// Write writes the underlying disk to the given writer.
func (o Operator) Write(w io.Writer) (int, error) {
	return o.SD.Write(w)
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

	op := Operator{SD: sd, SM: sm}

	st, err := sm.ReadSymbolTable(sd)
	if err == nil {
		op.ST = st
	}

	return op, nil
}

func init() {
	disk.RegisterDiskOperatorFactory(operatorName, operatorFactory)
}
