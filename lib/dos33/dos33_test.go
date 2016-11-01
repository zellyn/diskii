package dos33

import (
	"crypto/rand"
	"reflect"
	"testing"
)

// TestVTOCMarshalRoundtrip checks a simple roundtrip of VTOC data.
func TestVTOCMarshalRoundtrip(t *testing.T) {
	buf := make([]byte, 256)
	rand.Read(buf)
	vtoc1 := &VTOC{}
	if err := vtoc1.UnmarshalBinary(buf); err != nil {
		t.Fatal(err)
	}
	buf2, _ := vtoc1.MarshalBinary()
	if !reflect.DeepEqual(buf, buf2) {
		t.Errorf("Buffers differ: %v != %v", buf, buf2)
	}
	vtoc2 := &VTOC{}
	if err := vtoc2.UnmarshalBinary(buf2); err != nil {
		t.Fatal(err)
	}
	if *vtoc1 != *vtoc2 {
		t.Errorf("Structs differ: %v != %v", vtoc1, vtoc2)
	}
}

// TestCatalogSectorMarshalRoundtrip checks a simple roundtrip of CatalogSector data.
func TestCatalogSectorMarshalRoundtrip(t *testing.T) {
	buf := make([]byte, 256)
	rand.Read(buf)
	cs1 := &CatalogSector{}
	if err := cs1.UnmarshalBinary(buf); err != nil {
		t.Fatal(err)
	}
	buf2, _ := cs1.MarshalBinary()
	if !reflect.DeepEqual(buf, buf2) {
		t.Errorf("Buffers differ: %v != %v", buf, buf2)
	}
	cs2 := &CatalogSector{}
	if err := cs2.UnmarshalBinary(buf2); err != nil {
		t.Fatal(err)
	}
	if *cs1 != *cs2 {
		t.Errorf("Structs differ: %v != %v", cs1, cs2)
	}
}
