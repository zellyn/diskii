// Copyright © 2017 Zellyn Hunter <zellyn@gmail.com>

// Package prodos contains routines for working with the on-disk
// structures of Apple ProDOS.
package prodos

import (
	"fmt"
	"io"
)

// A single ProDOS block.
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
}

type VolumeBitMap []Block

func NewVolumeBitMap(blocks uint16) VolumeBitMap {
	vbm := VolumeBitMap(make([]Block, (blocks+(512*8)-1)/(512*8)))
	for b := 0; b < int(blocks); b++ {
		vbm.MarkUnused(uint16(b))
	}
	return vbm
}

func (vbm VolumeBitMap) MarkUsed(block uint16) {
	vbm.mark(block, false)
}

func (vbm VolumeBitMap) MarkUnused(block uint16) {
	vbm.mark(block, true)
}

func (vbm VolumeBitMap) mark(block uint16, set bool) {
	byteIndex := block >> 3
	blockIndex := byteIndex / 512
	blockByteIndex := byteIndex % 512
	bit := byte(1 << (7 - (block & 7)))
	if set {
		vbm[blockIndex][blockByteIndex] |= bit
	} else {
		vbm[blockIndex][blockByteIndex] &^= bit
	}
}

func (vbm VolumeBitMap) Free(block uint16) bool {
	byteIndex := block >> 3
	blockIndex := byteIndex / 512
	blockByteIndex := byteIndex % 512
	bit := byte(1 << (7 - (block & 7)))
	return vbm[blockIndex][blockByteIndex]&bit > 0
}

// ReadVolumeBitMap
func ReadVolumeBitMap(bd BlockDevice, startBlock uint16) (VolumeBitMap, error) {
	blocks := bd.Blocks() / 4096
	vbm := make([]Block, blocks)
	for i := uint16(0); i < blocks; i++ {
		block, err := bd.ReadBlock(startBlock + i)
		if err != nil {
			return nil, fmt.Errorf("cannot read block %d of Volume Bit Map: %v", err)
		}
		vbm[i] = block
	}
	return VolumeBitMap(vbm), nil
}

// Write writes the Volume Bit Map to a block device, starting at the
// given block.
func (vbm VolumeBitMap) Write(bd BlockDevice, startBlock uint16) error {
	for i, block := range vbm {
		if err := bd.WriteBlock(startBlock+uint16(i), block); err != nil {
			return fmt.Errorf("cannot write block %d of Volume Bit Map: %v", err)
		}
	}
	return nil
}

type DateTime struct {
	YMD [2]byte
	HM  [2]byte
}

// VolumeDirectoryKeyBlock is the struct used to hold the ProDOS Volume Directory Key
// Block structure.  See page 4-4 of Beneath Apple ProDOS.
type VolumeDirectoryKeyBlock struct {
	Prev        uint16 // Pointer to previous block (always zero: the KeyBlock is the first Volume Directory block
	Next        uint16 // Pointer to next block in the Volume Directory
	Header      VolumeDirectoryHeader
	Descriptors [12]FileDescriptor
}

// VolumeDirectoryBlock is a normal (non-key) segment in the Volume Directory Header.
type VolumeDirectoryBlock struct {
	Prev        uint16 // Pointer to previous block in the Volume Directory.
	Next        uint16 // Pointer to next block in the Volume Directory.
	Descriptors [13]FileDescriptor
}

type VolumeDirectoryHeader struct {
	TypeAndNameLength byte     // Storage type (top four bits) and volume name length (lower four).
	VolumeName        [15]byte // Volume name (actual length defined in TypeAndNameLength)
	Unused1           [8]byte
	Creation          DateTime // Date and time volume was formatted
	Version           byte
	MinVersion        byte
	Access            byte
	EntryLength       byte   // Length of each entry in the Volume Directory: usually $27
	EntriesPerBlock   byte   // Usually $0D
	FileCount         uint16 // Number of active entries in the Volume Directory, not counting the Volume Directory Header
	BitMapPointer     uint16 // Block number of start of VolumeBitMap. Usually 6
	TotalBlocks       uint16 // Total number of blocks on the device. $118 (280) for a 35-track diskette.
}

type Access byte

const (
	AccessReadable           Access = 0x01
	AccessWritable           Access = 0x02
	AccessChangedSinceBackup Access = 0x20
	AccessRenamable          Access = 0x40
	AccessDestroyable        Access = 0x80
)

// FileDescriptor is the entry in the volume directory for a file or
// subdirectory.
type FileDescriptor struct {
	TypeAndNameLength byte     // Storage type (top four bits) and volume name length (lower four)
	FileName          [15]byte // Filename (actual length defined in TypeAndNameLength)
	FileType          byte     // ProDOS / SOS filetype
	KeyPointer        uint16   // block number of key block for file
	BlocksUsed        uint16   // Total number of blocks used including index blocks and data blocks. For a subdirectory, the number of directory blocks
	Eof               [3]byte  // 3-byte offset of EOF from first byte. For sequential files, just the length
	Creation          DateTime // Date and time of of file creation
	Version           byte
	MinVersion        byte
	Access            Access
	// For TXT files, random access record length (L from OPEN)
	// For BIN files, load address for binary image (A from BSAVE)
	// For BAS files, load address for program image (when SAVEd)
	// For VAR files, address of compressed variables image (when STOREd)
	//  For SYS files, load address for system program (usually $2000)
	AuxType       uint16
	LastMod       DateTime
	HeaderPointer uint16 // Block  number of the key block for the directory which describes this file.
}

// An index block contains 256 16-bit block numbers, pointing to other
// blocks. The LSBs are stored in the first half, MSBs in the second.
type IndexBlock Block

// Get the blockNum'th block number from an index block.
func (i IndexBlock) Get(blockNum byte) uint16 {
	return uint16(i[blockNum]) + uint16(i[256+int(blockNum)])<<8
}

// Set the blockNum'th block number in an index block.
func (i IndexBlock) Set(blockNum byte, block uint16) {
	i[blockNum] = byte(block)
	i[256+int(blockNum)] = byte(block >> 8)
}
