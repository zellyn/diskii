package prodos

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/zellyn/diskii/lib/disk"
)

func randomBlock() disk.Block {
	var b1 disk.Block
	rand.Read(b1[:])
	return b1
}

// TestVolumeDirectoryKeyBlockMarshalRoundtrip checks a simple roundtrip of VDKB data.
func TestVolumeDirectoryKeyBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	vdkb := &VolumeDirectoryKeyBlock{}
	vdkb.FromBlock(b1)
	b2 := vdkb.ToBlock()
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	vdkb2 := &VolumeDirectoryKeyBlock{}
	vdkb2.FromBlock(b2)
	if *vdkb != *vdkb2 {
		t.Errorf("Structs differ: %v != %v", vdkb, vdkb2)
	}
}

// TestVolumeDirectoryBlockMarshalRoundtrip checks a simple roundtrip of VDB data.
func TestVolumeDirectoryBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	vdb := &VolumeDirectoryBlock{}
	vdb.FromBlock(b1)
	b2 := vdb.ToBlock()
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	vdb2 := &VolumeDirectoryBlock{}
	vdb2.FromBlock(b2)
	if *vdb != *vdb2 {
		t.Errorf("Structs differ: %v != %v", vdb, vdb2)
	}
}

// TestSubdirectoryKeyBlockMarshalRoundtrip checks a simple roundtrip of SKB data.
func TestSubdirectoryKeyBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	skb := &SubdirectoryKeyBlock{}
	skb.FromBlock(b1)
	b2 := skb.ToBlock()
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	skb2 := &SubdirectoryKeyBlock{}
	skb2.FromBlock(b2)
	if *skb != *skb2 {
		t.Errorf("Structs differ: %v != %v", skb, skb2)
	}
}

// TestSubdirectoryBlockMarshalRoundtrip checks a simple roundtrip of SB data.
func TestSubdirectoryBlockMarshalRoundtrip(t *testing.T) {
	b1 := randomBlock()
	sb := &SubdirectoryBlock{}
	sb.FromBlock(b1)
	b2 := sb.ToBlock()
	if b1 != b2 {
		t.Fatalf("Blocks differ: %s", strings.Join(pretty.Diff(b1[:], b2[:]), "; "))
	}
	sb2 := &SubdirectoryBlock{}
	sb2.FromBlock(b2)
	if *sb != *sb2 {
		t.Errorf("Structs differ: %v != %v", sb, sb2)
	}
}
