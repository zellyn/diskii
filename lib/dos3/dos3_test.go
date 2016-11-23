package dos3

import (
	"crypto/rand"
	"reflect"
	"testing"

	"github.com/zellyn/diskii/lib/disk"
)

// TestVTOCMarshalRoundtrip checks a simple roundtrip of VTOC data.
func TestVTOCMarshalRoundtrip(t *testing.T) {
	buf := make([]byte, 256)
	rand.Read(buf)
	buf1 := make([]byte, 256)
	copy(buf1, buf)
	vtoc1 := &VTOC{}
	vtoc1.FromSector(buf1)
	buf2, err := vtoc1.ToSector()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, buf2) {
		t.Errorf("Buffers differ: %v != %v", buf, buf2)
	}
	vtoc2 := &VTOC{}
	vtoc2.FromSector(buf2)
	if *vtoc1 != *vtoc2 {
		t.Errorf("Structs differ: %v != %v", vtoc1, vtoc2)
	}
}

// TestCatalogSectorMarshalRoundtrip checks a simple roundtrip of CatalogSector data.
func TestCatalogSectorMarshalRoundtrip(t *testing.T) {
	buf := make([]byte, 256)
	rand.Read(buf)
	buf1 := make([]byte, 256)
	copy(buf1, buf)
	cs1 := &CatalogSector{}
	cs1.FromSector(buf1)
	buf2, err := cs1.ToSector()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, buf2) {
		t.Errorf("Buffers differ: %v != %v", buf, buf2)
	}
	cs2 := &CatalogSector{}
	cs2.FromSector(buf2)
	if *cs1 != *cs2 {
		t.Errorf("Structs differ: %v != %v", cs1, cs2)
	}
}

// TestTrackSectorListMarshalRoundtrip checks a simple roundtrip of TrackSectorList data.
func TestTrackSectorListMarshalRoundtrip(t *testing.T) {
	buf := make([]byte, 256)
	rand.Read(buf)
	buf1 := make([]byte, 256)
	copy(buf1, buf)
	cs1 := &TrackSectorList{}
	cs1.FromSector(buf1)
	buf2, err := cs1.ToSector()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(buf, buf2) {
		t.Errorf("Buffers differ: %v != %v", buf, buf2)
	}
	cs2 := &TrackSectorList{}
	cs2.FromSector(buf2)
	if *cs1 != *cs2 {
		t.Errorf("Structs differ: %v != %v", cs1, cs2)
	}
}

// TestReadCatalog tests the reading of the catalog of a test disk.
func TestReadCatalog(t *testing.T) {
	sd, err := disk.LoadDSK("testdata/dos33test.dsk")
	if err != nil {
		t.Fatal(err)
	}
	dsk, err := disk.NewMappedDisk(sd, disk.Dos33LogicalToPhysicalSectorMap)
	if err != nil {
		t.Fatal(err)
	}
	fds, deleted, err := ReadCatalog(dsk)

	fdsWant := []struct {
		locked bool
		typ    string
		size   int
		name   string
	}{
		{true, "A", 3, "HELLO"},
		{true, "I", 3, "APPLESOFT"},
		{true, "B", 6, "LOADER.OBJ0"},
		{true, "B", 42, "FPBASIC"},
		{true, "B", 42, "INTBASIC"},
		{true, "A", 3, "MASTER"},
		{true, "B", 9, "MASTER CREATE"},
		{true, "I", 9, "COPY"},
		{true, "B", 3, "COPY.OBJ0"},
		{true, "A", 9, "COPYA"},
		{true, "B", 3, "CHAIN"},
		{true, "A", 14, "RENUMBER"},
		{true, "A", 3, "FILEM"},
		{true, "B", 20, "FID"},
		{true, "A", 3, "CONVERT13"},
		{true, "B", 27, "MUFFIN"},
		{true, "A", 3, "START13"},
		{true, "B", 7, "BOOT13"},
		{true, "A", 4, "SLOT#"},
		{false, "A", 3, "EXAMPLE"},
		{false, "I", 2, "EXAMPLE2"},
		{false, "I", 2, "EXAMPLE3"},
	}

	deletedWant := []struct {
		locked bool
		typ    string
		size   int
		name   string
	}{
		{false, "I", 3, "EXAMPLE4"},
		{false, "A", 3, "EXAMPLE5"},
	}

	if len(fdsWant) != len(fds) {
		t.Fatalf("Want %d undeleted files; got %d", len(fdsWant), len(fds))
	}

	if len(deletedWant) != len(deleted) {
		t.Fatalf("Want %d deleted files; got %d", len(deletedWant), len(deleted))
	}

	for i, wantInfo := range fdsWant {
		if want, got := wantInfo.name, fds[i].FilenameString(); want != got {
			t.Errorf("Want filename %d to be %q; got %q", i+1, want, got)
		}
	}

	for i, wantInfo := range deletedWant {
		if want, got := wantInfo.name, deleted[i].FilenameString(); want != got {
			t.Errorf("Want deleted filename %d to be %q; got %q", i+1, want, got)
		}
	}

	// TODO(zellyn): Check type, size, locked status.
}
