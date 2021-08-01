package prodos

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/zellyn/diskii/disk"
)

func randomBlock() disk.Block {
	var b1 disk.Block
	_, _ = rand.Read(b1[:])
	return b1
}

// TestVolumeDirectoryKeyBlockMarshalRoundtrip checks a simple roundtrip of VDKB data.
func TestVolumeDirectoryKeyBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	vdkb := &VolumeDirectoryKeyBlock{}
	err := vdkb.FromBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := vdkb.ToBlock()
	if err != nil {
		t.Fatal(err)
	}
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	vdkb2 := &VolumeDirectoryKeyBlock{}
	err = vdkb2.FromBlock(b2)
	if err != nil {
		t.Fatal(err)
	}
	if *vdkb != *vdkb2 {
		t.Errorf("Structs differ: %v != %v", vdkb, vdkb2)
	}
}

// TestVolumeDirectoryBlockMarshalRoundtrip checks a simple roundtrip of VDB data.
func TestVolumeDirectoryBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	vdb := &VolumeDirectoryBlock{}
	err := vdb.FromBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := vdb.ToBlock()
	if err != nil {
		t.Fatal(err)
	}
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	vdb2 := &VolumeDirectoryBlock{}
	err = vdb2.FromBlock(b2)
	if err != nil {
		t.Fatal(err)
	}
	if *vdb != *vdb2 {
		t.Errorf("Structs differ: %v != %v", vdb, vdb2)
	}
}

// TestSubdirectoryKeyBlockMarshalRoundtrip checks a simple roundtrip of SKB data.
func TestSubdirectoryKeyBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	skb := &SubdirectoryKeyBlock{}
	err := skb.FromBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := skb.ToBlock()
	if err != nil {
		t.Fatal(err)
	}
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	skb2 := &SubdirectoryKeyBlock{}
	err = skb2.FromBlock(b2)
	if err != nil {
		t.Fatal(err)
	}
	if *skb != *skb2 {
		t.Errorf("Structs differ: %v != %v", skb, skb2)
	}
}

// TestSubdirectoryBlockMarshalRoundtrip checks a simple roundtrip of SB data.
func TestSubdirectoryBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	sb := &SubdirectoryBlock{}
	err := sb.FromBlock(b1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := sb.ToBlock()
	if err != nil {
		t.Fatal(err)
	}
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	sb2 := &SubdirectoryBlock{}
	err = sb2.FromBlock(b2)
	if err != nil {
		t.Fatal(err)
	}
	if *sb != *sb2 {
		t.Errorf("Structs differ: %v != %v", sb, sb2)
	}
}
