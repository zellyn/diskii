// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package supermon

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

const testDisk = "testdata/chacha20.dsk"

const cities = `It was the best of times, it was the worst of times, it was the age of wisdom, it was the age of foolishness, it was the epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of Darkness, it was the spring of hope, it was the winter of despair, we had everything before us, we had nothing before us, we were all going direct to Heaven, we were all going direct the other way - in short, the period was so far like the present period, that some of its noisiest authorities insisted on its being received, for good or for evil, in the superlative degree of comparison only.`

// The extra newline pads us to 256 bytes.
const hamlet = `To be, or not to be, that is the question:
Whether 'tis Nobler in the mind to suffer
The Slings and Arrows of outrageous Fortune,
Or to take Arms against a Sea of troubles,
And by opposing end them: to die, to sleep
No more; and by a sleep, to say we end

`

// loadSectorMap loads a sector map for the disk image contained in
// filename. It returns the sector map and a sector disk.
func loadSectorMap(filename string) (SectorMap, []byte, error) {
	rawbytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}
	diskbytes, err := disk.Swizzle(rawbytes, disk.Dos33LogicalToPhysicalSectorMap)
	if err != nil {
		return nil, nil, err
	}
	sm, err := LoadSectorMap(diskbytes)
	if err != nil {
		return nil, nil, err
	}
	return sm, diskbytes, nil
}

// TestReadSectorMap tests the reading of the sector map of a test
// disk.
func TestReadSectorMap(t *testing.T) {
	sm, _, err := loadSectorMap(testDisk)
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.Verify(); err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		file   byte
		length int
		name   string
	}{
		{1, 0x02, "FHELLO"},
		{2, 0x17, "FSUPERMON"},
		{3, 0x10, "FSYMTBL1"},
		{4, 0x10, "FSYMTBL2"},
		{5, 0x1E, "FMONHELP"},
		{6, 0x08, "FSHORTSUP"},
		{7, 0x1F, "FSHRTHELP"},
		{8, 0x04, "FSHORT"},
		{9, 0x60, "FCHACHA"},
		{10, 0x01, "FTOBE"},
	}

	sectorsByFile := sm.SectorsByFile()
	for _, tt := range testData {
		sectors := sectorsByFile[tt.file]
		if len(sectors) != tt.length {
			t.Errorf("Want %q to be %d sectors long; got %d", tt.name, tt.length, len(sectors))
		}
	}
}

// TestReadSymbolTable tests the reading of the symbol table of a test
// disk.
func TestReadSymbolTable(t *testing.T) {
	sm, sd, err := loadSectorMap(testDisk)
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.Verify(); err != nil {
		t.Fatal(err)
	}

	st, err := sm.ReadSymbolTable(sd)
	if err != nil {
		t.Fatal(err)
	}
	symbols := st.SymbolsByAddress()

	testData := []struct {
		file uint16
		name string
	}{
		{1, "FHELLO"},
		{2, "FSUPERMON"},
		{3, "FSYMTBL1"},
		{4, "FSYMTBL2"},
		{5, "FMONHELP"},
		{6, "FSHORTSUP"},
		{7, "FSHRTHELP"},
		{8, "FSHORT"},
		{9, "FCHACHA"},
		{10, "FTOBE"},
	}

	for _, tt := range testData {
		fileAddr := uint16(0xDF00) + tt.file
		syms := symbols[fileAddr]
		if len(syms) != 1 {
			t.Errorf("Expected one symbol for address %04X (file %q), but got %d.", fileAddr, tt.file, len(syms))
			continue
		}
		if syms[0].Name != tt.name {
			t.Errorf("Expected symbol name for address %04X to be %q, but got %q.", fileAddr, tt.name, syms[0].Name)
			continue
		}
	}
}

// TestGetFile tests the retrieval of a file's contents, using the
// Operator interface.
func TestGetFile(t *testing.T) {
	op, _, err := disk.OpenFilename(testDisk, types.DiskOrderAuto, "nakedos", []types.OperatorFactory{OperatorFactory{}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	file, err := op.GetFile("FTOBE")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := hamlet, string(file.Data); got != want {
		t.Errorf("Incorrect result for GetFile(\"TOBE\"): want %q; got %q", want, got)
	}
}

// TestEncodeDecode tests encoding and decoding of Super-Mon symbol
// table entries.
func TestEncodeDecode(t *testing.T) {
	testdata := []struct {
		sym   string
		valid bool
	}{
		{"", true},
		{"ABC", true},
		{"abc", true},
		{"ABCDEFGHI", true},
		{"abcdefghi", true},
		{"ABCDEF123", true},
		{"abcdef123", true},

		{"AB", false},
		{"ab", false},
		{"ABCDE1234", false},
		{"abcde1234", false},
	}

	for _, tt := range testdata {
		if !tt.valid {
			if _, err := encodeSymbol(tt.sym); err == nil {
				t.Errorf("Expected symbol %q to be invalid, but wasn't", tt.sym)
			}
			continue
		}

		bytes, err := encodeSymbol(tt.sym)
		if err != nil {
			t.Errorf("Unexpected error encoding symbol %q", tt.sym)
			continue
		}
		sym := decodeSymbol(bytes[:5], bytes[5])
		if sym != strings.ToUpper(tt.sym) {
			t.Errorf("Symbol %q encodes to %q", tt.sym, sym)
		}
	}
}

// TestReadWriteSymbolTable tests reading, writing, and re-reading of
// the symbol table, ensuring that no details are lost along the way.
func TestReadWriteSymbolTable(t *testing.T) {
	sm, sd, err := loadSectorMap(testDisk)
	if err != nil {
		t.Fatal(err)
	}
	st1, err := sm.ReadSymbolTable(sd)
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.WriteSymbolTable(sd, st1); err != nil {
		t.Fatal(err)
	}
	st2, err := sm.ReadSymbolTable(sd)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(st1, st2) {
		pretty.Ldiff(t, st1, st2)
		t.Fatal("Saved and reloaded symbol table differs from original symbol table")
	}
}

// TestPutFile tests the creation of a file, using the Operator
// interface.
func TestPutFile(t *testing.T) {
	op, _, err := disk.OpenFilename(testDisk, types.DiskOrderAuto, "nakedos", []types.OperatorFactory{OperatorFactory{}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	contents := []byte(cities)
	fileInfo := types.FileInfo{
		Descriptor: types.Descriptor{
			Name:   "FNEWFILE",
			Length: len(contents),
			Type:   types.FiletypeBinary,
		},
		Data: contents,
	}
	existed, err := op.PutFile(fileInfo, false)
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Errorf("want existed=%v; got %v", false, existed)
	}

	fds, err := op.Catalog("")
	if err != nil {
		t.Fatal(err)
	}
	last := fds[len(fds)-1]
	if want, got := "DF0B:FNEWFILE", last.Fullname; got != want {
		t.Fatalf("Want last file on disk's FullName=%q; got %q", want, got)
	}
}
