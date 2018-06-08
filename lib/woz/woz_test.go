package woz_test

import (
	"bytes"
	"testing"

	"github.com/zellyn/diskii/data"
	"github.com/zellyn/diskii/lib/woz"
)

func TestBasicLoad(t *testing.T) {
	bb := data.MustAsset("data/disks/dos33master.woz")
	wz, err := woz.Decode(bytes.NewReader(bb))
	if err != nil {
		t.Fatal(err)
	}
	if len(wz.Unknowns) > 0 {
		t.Fatalf("want 0 unknowns; got %d", len(wz.Unknowns))
	}
	// fmt.Printf("%#v\n", wz)
	// t.Fatal("NOTHING")
}
