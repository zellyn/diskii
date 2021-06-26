// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// marshal.go contains helpers for marshaling sector structs to/from
// disk and block structs to/from devices.

package disk

import "fmt"

// BlockDevice is the interface used to read and write devices by
// logical block number.

// SectorSource is the interface for types that can marshal to sectors.
type SectorSource interface {
	// ToSector marshals the sector struct to exactly 256 bytes.
	ToSector() ([]byte, error)
	// GetTrack returns the track that a sector struct was loaded from.
	GetTrack() byte
	// GetSector returns the sector that a sector struct was loaded from.
	GetSector() byte
}

// SectorSink is the interface for types that can unmarshal from sectors.
type SectorSink interface {
	// FromSector unmarshals the sector struct from bytes. Input is
	// expected to be exactly 256 bytes.
	FromSector(data []byte) error
	// SetTrack sets the track that a sector struct was loaded from.
	SetTrack(track byte)
	// SetSector sets the sector that a sector struct was loaded from.
	SetSector(sector byte)
}

// UnmarshalLogicalSector reads a sector from a disk image, and unmarshals it
// into a SectorSink, setting its track and sector.
func UnmarshalLogicalSector(diskbytes []byte, ss SectorSink, track, sector byte) error {
	bytes, err := ReadSector(diskbytes, track, sector)
	if err != nil {
		return err
	}
	if err := ss.FromSector(bytes); err != nil {
		return err
	}
	ss.SetTrack(track)
	ss.SetSector(sector)
	return nil
}

// ReadSector just reads 256 bytes from the given track and sector.
func ReadSector(diskbytes []byte, track, sector byte) ([]byte, error) {
	start := int(track)*FloppyTrackBytes + int(sector)*256
	end := start + 256
	if len(diskbytes) < end {
		return nil, fmt.Errorf("cannot read track %d/sector %d (bytes %d-%d) from disk of length %d", track, sector, start, end, len(diskbytes))
	}
	bytes := make([]byte, 256)
	copy(bytes, diskbytes[start:end])
	return bytes, nil
}

// MarshalLogicalSector marshals a SectorSource to its track/sector on a disk
// image.
func MarshalLogicalSector(diskbytes []byte, ss SectorSource) error {
	track := ss.GetTrack()
	sector := ss.GetSector()
	bytes, err := ss.ToSector()
	if err != nil {
		return err
	}
	return WriteSector(diskbytes, track, sector, bytes)
}

// WriteSector writes 256 bytes to the given track and sector.
func WriteSector(diskbytes []byte, track, sector byte, data []byte) error {
	if len(data) != 256 {
		return fmt.Errorf("call to writeSector with len(data)==%d; want 256", len(data))
	}
	start := int(track)*FloppyTrackBytes + int(sector)*256
	end := start + 256
	if len(diskbytes) < end {
		return fmt.Errorf("cannot write track %d/sector %d (bytes %d-%d) to disk of length %d", track, sector, start, end, len(diskbytes))
	}
	copy(diskbytes[start:end], data)
	return nil
}

// BlockSource is the interface for types that can marshal to blocks.
type BlockSource interface {
	// ToBlock marshals the block struct to exactly 512 bytes.
	ToBlock() (Block, error)
	// GetBlock returns the index that a block struct was loaded from.
	GetBlock() uint16
}

// BlockSink is the interface for types that can unmarshal from blocks.
type BlockSink interface {
	// FromBlock unmarshals the block struct from a Block. Input is
	// expected to be exactly 512 bytes.
	FromBlock(block Block) error
	// SetBlock sets the index that a block struct was loaded from.
	SetBlock(index uint16)
}

// UnmarshalBlock reads a block from a block device, and unmarshals it into a
// BlockSink, setting its index.
func UnmarshalBlock(diskbytes []byte, bs BlockSink, index uint16) error {
	start := int(index) * 512
	end := start + 512
	if len(diskbytes) < end {
		return fmt.Errorf("device too small to read block %d", index)
	}
	var block Block
	copy(block[:], diskbytes[start:end])
	if err := bs.FromBlock(block); err != nil {
		return err
	}
	bs.SetBlock(index)
	return nil
}

// MarshalBlock marshals a BlockSource to its block on a block device.
func MarshalBlock(diskbytes []byte, bs BlockSource) error {
	index := bs.GetBlock()
	block, err := bs.ToBlock()
	if err != nil {
		return err
	}
	start := int(index) * 512
	end := start + 512
	if len(diskbytes) < end {
		return fmt.Errorf("device too small to write block %d", index)
	}
	copy(diskbytes[start:end], block[:])
	return nil
}
