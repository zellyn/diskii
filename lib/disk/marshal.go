// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// marshal.go contains helpers for marshaling sector structs to/from
// disk and block structs to/from devices.

package disk

import "io"

// A ProDOS block.
type Block [512]byte

// BlockDevice is the interface used to read and write devices by
// logical block number.
type BlockDevice interface {
	// ReadBlock reads a single block from the device. It always returns
	// 512 byes.
	ReadBlock(index uint16) (Block, error)
	// WriteBlock writes a single block to a device. It expects exactly
	// 512 bytes.
	WriteBlock(index uint16, data Block) error
	// Blocks returns the number of blocks on the device.
	Blocks() uint16
	// Write writes the device contents to the given Writer.
	Write(io.Writer) (int, error)
	// Order returns the sector or block order of the underlying device.
	Order() string
}

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

// UnmarshalLogicalSector reads a sector from a SectorDisk, and
// unmarshals it into a SectorSink, setting its track and sector.
func UnmarshalLogicalSector(d LogicalSectorDisk, ss SectorSink, track, sector byte) error {
	bytes, err := d.ReadLogicalSector(track, sector)
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

// MarshalLogicalSector marshals a SectorSource to its sector on a
// SectorDisk.
func MarshalLogicalSector(d LogicalSectorDisk, ss SectorSource) error {
	track := ss.GetTrack()
	sector := ss.GetSector()
	bytes, err := ss.ToSector()
	if err != nil {
		return err
	}
	return d.WriteLogicalSector(track, sector, bytes)
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

// UnmarshalBlock reads a block from a BlockDevice, and unmarshals it
// into a BlockSink, setting its index.
func UnmarshalBlock(d BlockDevice, bs BlockSink, index uint16) error {
	block, err := d.ReadBlock(index)
	if err != nil {
		return err
	}
	if err := bs.FromBlock(block); err != nil {
		return err
	}
	bs.SetBlock(index)
	return nil
}

// MarshalBlock marshals a BlockSource to its block on a BlockDevice.
func MarshalBlock(d BlockDevice, bs BlockSource) error {
	index := bs.GetBlock()
	block, err := bs.ToBlock()
	if err != nil {
		return err
	}
	return d.WriteBlock(index, block)
}
