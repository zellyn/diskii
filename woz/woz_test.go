package woz_test

import (
	"bytes"
	"testing"

	"github.com/zellyn/diskii/data"
	"github.com/zellyn/diskii/woz"
)

func TestBasicLoad(t *testing.T) {
	wz, err := woz.Decode(bytes.NewReader(data.DOS33master_woz))
	if err != nil {
		t.Fatal(err)
	}
	if len(wz.Unknowns) > 0 {
		t.Fatalf("want 0 unknowns; got %d", len(wz.Unknowns))
	}
	// fmt.Printf("%#v\n", wz)
	// t.Fatal("NOTHING")
}
