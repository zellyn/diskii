// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package supermon

import (
	"testing"

	"github.com/zellyn/diskii/lib/disk"
)

// loadSectorMap loads a sector map for the disk image contained in
// filename.
func loadSectorMap(filename string) (SectorMap, error) {
	dsk, err := disk.LoadDSK(filename)
	if err != nil {
		return nil, err
	}
	sd := SectorDiskShim{Dos33: dsk}
	sm, err := LoadSectorMap(sd)
	if err != nil {
		return nil, err
	}
	return sm, nil
}

// TestReadSectorMap tests the reading of the sector map of a test
// disk.
func TestReadSectorMap(t *testing.T) {
	sm, err := loadSectorMap("testdata/chacha20.dsk")
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
	}

	sectorsByFile := sm.SectorsByFile()
	for _, tt := range testData {
		sectors := sectorsByFile[tt.file]
		if len(sectors) != tt.length {
			t.Errorf("Want %q to be %d sectors long; got %d", tt.name, tt.length, len(sectors))
		}
	}
}
