// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package supermon

import (
	"testing"

	"github.com/zellyn/diskii/lib/disk"
)

const testDisk = "testdata/chacha20.dsk"

// loadSectorMap loads a sector map for the disk image contained in
// filename. It returns the sector map and a sector disk.
func loadSectorMap(filename string) (SectorMap, disk.SectorDisk, error) {
	sd, err := disk.LoadDSK(filename)
	if err != nil {
		return nil, nil, err
	}
	sm, err := LoadSectorMap(sd)
	if err != nil {
		return nil, nil, err
	}
	return sm, sd, nil
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
	sd, err := disk.Open(testDisk)
	if err != nil {
		t.Fatal(err)
	}
	op, err := disk.OperatorFor(sd)
	if err != nil {
		t.Fatal(err)
	}
	file, err := op.GetFile("FTOBE")
	if err != nil {
		t.Fatal(err)
	}
	got := string(file.Data)
	// The extra newline pads us to 256 bytes…
	want := `To be, or not to be, that is the question:
Whether 'tis Nobler in the mind to suffer
The Slings and Arrows of outrageous Fortune,
Or to take Arms against a Sea of troubles,
And by opposing end them: to die, to sleep
No more; and by a sleep, to say we end

`
	if got != want {
		t.Errorf("Incorrect result for GetFile(\"TOBE\"): want %q; got %q", want, got)
	}
}
