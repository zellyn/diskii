// Copyright © 2017 Zellyn Hunter <zellyn@gmail.com>

// Package prodos contains routines for working with the on-disk
// structures of Apple ProDOS.
//
// TODO(zellyn): remove errors from FromBlock(), and move validation
// into separate Validate() functions.
package prodos

import (
	"encoding/binary"
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

// IsFree returns true if the given block on the device is free,
// according to the VolumeBitMap.
func (vbm VolumeBitMap) IsFree(block uint16) bool {
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

// DateTime represents the 4-byte ProDOS y/m/d h/m timestamp.
type DateTime struct {
	YMD [2]byte
	HM  [2]byte
}

// toBytes returns a four-byte slice representing a DateTime.
func (dt DateTime) toBytes() []byte {
	return []byte{dt.YMD[0], dt.YMD[1], dt.HM[0], dt.HM[1]}
}

// fromBytes turns a slice of four bytes back into a DateTime.
func (dt *DateTime) fromBytes(b []byte) error {
	if len(b) != 4 {
		return fmt.Errorf("DateTime expects 4 bytes; got %d", len(b))
	}
	if b[2] >= 0x20 {
		return fmt.Errorf("DateTime expects hour<0x20; got 0x%02x", b[2])
	}
	if b[3] >= 0x40 {
		return fmt.Errorf("DateTime expects minute<0x40; got 0x%02x", b[3])
	}
	dt.YMD[0] = b[0]
	dt.YMD[1] = b[1]
	dt.HM[0] = b[2]
	dt.HM[1] = b[3]
	return nil
}

// VolumeDirectoryKeyBlock is the struct used to hold the ProDOS Volume Directory Key
// Block structure.  See page 4-4 of Beneath Apple ProDOS.
type VolumeDirectoryKeyBlock struct {
	Prev        uint16 // Pointer to previous block (always zero: the KeyBlock is the first Volume Directory block
	Next        uint16 // Pointer to next block in the Volume Directory
	Header      VolumeDirectoryHeader
	Descriptors [12]FileDescriptor
}

// ToBlock marshals the VolumeDirectoryKeyBlock to a Block of bytes.
func (vdkb VolumeDirectoryKeyBlock) ToBlock() Block {
	var block Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], vdkb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], vdkb.Next)
	copyBytes(block[0x04:0x02b], vdkb.Header.toBytes())
	for i, desc := range vdkb.Descriptors {
		copyBytes(block[0x2b+i*0x27:0x2b+(i+1)*0x27], desc.toBytes())
	}
	return block
}

// FromBlock unmarshals a Block of bytes into a VolumeDirectoryKeyBlock.
func (vdkb *VolumeDirectoryKeyBlock) FromBlock(block Block) error {
	vdkb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	if vdkb.Prev != 0 {
		return fmt.Errorf("Volume Directory Key Block should have a `Previous` block of 0, got $%04x", vdkb.Prev)
	}
	vdkb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	if err := vdkb.Header.fromBytes(block[0x04:0x2b]); err != nil {
		return err
	}
	for i := range vdkb.Descriptors {
		if err := vdkb.Descriptors[i].fromBytes(block[0x2b+i*0x27 : 0x2b+(i+1)*0x27]); err != nil {
			return fmt.Errorf("cannot deserialize file descriptor %d of Volume Directory Key Block: %v", err)
		}
	}
	return nil
}

// VolumeDirectoryBlock is a normal (non-key) segment in the Volume Directory Header.
type VolumeDirectoryBlock struct {
	Prev        uint16 // Pointer to previous block in the Volume Directory.
	Next        uint16 // Pointer to next block in the Volume Directory.
	Descriptors [13]FileDescriptor
}

// ToBlock marshals a VolumeDirectoryBlock to a Block of bytes.
func (vdb VolumeDirectoryBlock) ToBlock() Block {
	var block Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], vdb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], vdb.Next)
	for i, desc := range vdb.Descriptors {
		copyBytes(block[0x04+i*0x27:0x04+(i+1)*0x27], desc.toBytes())
	}
	return block
}

// FromBlock unmarshals a Block of bytes into a VolumeDirectoryBlock.
func (vdb *VolumeDirectoryBlock) FromBlock(block Block) error {
	vdb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	vdb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	for i := range vdb.Descriptors {
		if err := vdb.Descriptors[i].fromBytes(block[0x4+i*0x27 : 0x4+(i+1)*0x27]); err != nil {
			return fmt.Errorf("cannot deserialize file descriptor %d of Volume Directory Block: %v", err)
		}
	}
	return nil
}

type VolumeDirectoryHeader struct {
	TypeAndNameLength byte     // Storage type (top four bits) and volume name length (lower four).
	VolumeName        [15]byte // Volume name (actual length defined in TypeAndNameLength)
	Unused1           [8]byte
	Creation          DateTime // Date and time volume was formatted
	Version           byte
	MinVersion        byte
	Access            Access
	EntryLength       byte   // Length of each entry in the Volume Directory: usually $27
	EntriesPerBlock   byte   // Usually $0D
	FileCount         uint16 // Number of active entries in the Volume Directory, not counting the Volume Directory Header
	BitMapPointer     uint16 // Block number of start of VolumeBitMap. Usually 6
	TotalBlocks       uint16 // Total number of blocks on the device. $118 (280) for a 35-track diskette.
}

// toBytes converts a VolumeDirectoryHeader to a slice of bytes.
func (vdh VolumeDirectoryHeader) toBytes() []byte {
	buf := make([]byte, 0x27)
	buf[0] = vdh.TypeAndNameLength
	copyBytes(buf[1:0x10], vdh.VolumeName[:])
	copyBytes(buf[0x10:0x18], vdh.Unused1[:])
	copyBytes(buf[0x18:0x1c], vdh.Creation.toBytes())
	buf[0x1c] = vdh.Version
	buf[0x1d] = vdh.MinVersion
	buf[0x1e] = byte(vdh.Access)
	buf[0x1f] = vdh.EntryLength
	buf[0x20] = vdh.EntriesPerBlock
	binary.LittleEndian.PutUint16(buf[0x20:0x22], vdh.FileCount)
	binary.LittleEndian.PutUint16(buf[0x22:0x24], vdh.BitMapPointer)
	binary.LittleEndian.PutUint16(buf[0x24:0x26], vdh.TotalBlocks)
	return buf
}

// fromBytes unmarshals a slice of bytes into a VolumeDirectoryHeader.
func (vdh *VolumeDirectoryHeader) fromBytes(buf []byte) error {
	if len(buf) != 0x27 {
		return fmt.Errorf("VolumeDirectoryHeader should be 0x27 bytes long; got 0x%02x", len(buf))
	}
	vdh.TypeAndNameLength = buf[0]
	copyBytes(vdh.VolumeName[:], buf[1:0x10])
	copyBytes(vdh.Unused1[:], buf[0x10:0x18])
	if err := vdh.Creation.fromBytes(buf[0x18:0x1c]); err != nil {
		return fmt.Errorf("unable to deserialize Volume Directory Header Creation date/time: %v", err)
	}
	vdh.Version = buf[0x1c]
	vdh.MinVersion = buf[0x1d]
	vdh.Access = Access(buf[0x1e])
	vdh.EntryLength = buf[0x1f]
	vdh.EntriesPerBlock = buf[0x20]
	vdh.FileCount = binary.LittleEndian.Uint16(buf[0x20:0x22])
	vdh.BitMapPointer = binary.LittleEndian.Uint16(buf[0x22:0x24])
	vdh.TotalBlocks = binary.LittleEndian.Uint16(buf[0x24:0x26])
	return nil
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

// Filename returns the string filename of a file descriptor.
func (fd FileDescriptor) Filename() string {
	return string(fd.FileName[0 : fd.TypeAndNameLength&0xf])
}

// toBytes converts a FileDescriptor to a slice of bytes.
func (fd FileDescriptor) toBytes() []byte {
	buf := make([]byte, 0x27)
	buf[0] = fd.TypeAndNameLength
	copyBytes(buf[1:0x10], fd.FileName[:])
	buf[0x10] = fd.FileType
	binary.LittleEndian.PutUint16(buf[0x11:0x13], fd.KeyPointer)
	binary.LittleEndian.PutUint16(buf[0x13:0x15], fd.BlocksUsed)
	copyBytes(buf[0x15:0x18], fd.Eof[:])
	copyBytes(buf[0x18:0x1c], fd.Creation.toBytes())
	buf[0x1c] = fd.Version
	buf[0x1d] = fd.MinVersion
	buf[0x1e] = byte(fd.Access)
	binary.LittleEndian.PutUint16(buf[0x1f:0x21], fd.AuxType)
	copyBytes(buf[0x21:0x25], fd.LastMod.toBytes())
	binary.LittleEndian.PutUint16(buf[0x25:0x27], fd.HeaderPointer)
	return buf
}

// fromBytes unmarshals a slice of bytes into a FileDescriptor.
func (fd *FileDescriptor) fromBytes(buf []byte) error {
	if len(buf) != 0x27 {
		return fmt.Errorf("FileDescriptor should be 0x27 bytes long; got 0x%02x", len(buf))
	}
	fd.TypeAndNameLength = buf[0]
	copyBytes(fd.FileName[:], buf[1:0x10])
	fd.FileType = buf[0x10]

	fd.KeyPointer = binary.LittleEndian.Uint16(buf[0x11:0x13])
	fd.BlocksUsed = binary.LittleEndian.Uint16(buf[0x13:0x15])
	copyBytes(fd.Eof[:], buf[0x15:0x18])
	if err := fd.Creation.fromBytes(buf[0x18:0x1c]); err != nil {
		return fmt.Errorf("unable to unmarshal Creation date/time of FileDescriptor %q: %v", fd.Filename(), err)
	}
	fd.Version = buf[0x1c]
	fd.MinVersion = buf[0x1d]
	fd.Access = Access(buf[0x1e])
	fd.AuxType = binary.LittleEndian.Uint16(buf[0x1f:0x21])
	if err := fd.LastMod.fromBytes(buf[0x21:0x25]); err != nil {
		return fmt.Errorf("unable to unmarshal last modification date/time of FileDescriptor %q: %v", fd.Filename(), err)
	}
	fd.HeaderPointer = binary.LittleEndian.Uint16(buf[0x25:0x27])

	return nil
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

// SubdirectoryKeyBlock is the struct used to hold the first entry in
// a subdirectory structure.
type SubdirectoryKeyBlock struct {
	Prev        uint16 // Pointer to previous block (always zero: the KeyBlock is the first Volume Directory block
	Next        uint16 // Pointer to next block in the Volume Directory
	Header      SubdirectoryHeader
	Descriptors [12]FileDescriptor
}

// ToBlock marshals the SubdirectoryKeyBlock to a Block of bytes.
func (skb SubdirectoryKeyBlock) ToBlock() Block {
	var block Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], skb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], skb.Next)
	copyBytes(block[0x04:0x02b], skb.Header.toBytes())
	for i, desc := range skb.Descriptors {
		copyBytes(block[0x2b+i*0x27:0x2b+(i+1)*0x27], desc.toBytes())
	}
	return block
}

// FromBlock unmarshals a Block of bytes into a SubdirectoryKeyBlock.
func (skb *SubdirectoryKeyBlock) FromBlock(block Block) error {
	skb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	if skb.Prev != 0 {
		return fmt.Errorf("Subdirectory Key Block should have a `Previous` block of 0, got $%04x", skb.Prev)
	}
	skb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	if err := skb.Header.fromBytes(block[0x04:0x2b]); err != nil {
		return err
	}
	for i := range skb.Descriptors {
		if err := skb.Descriptors[i].fromBytes(block[0x2b+i*0x27 : 0x2b+(i+1)*0x27]); err != nil {
			return fmt.Errorf("cannot deserialize file descriptor %d of Subdirectory Key Block: %v", err)
		}
	}
	return nil
}

// SubdirectoryBlock is a normal (non-key) segment in a Subdirectory.
type SubdirectoryBlock struct {
	Prev        uint16 // Pointer to previous block in the Volume Directory.
	Next        uint16 // Pointer to next block in the Volume Directory.
	Descriptors [13]FileDescriptor
}

// ToBlock marshals a SubdirectoryBlock to a Block of bytes.
func (sb SubdirectoryBlock) ToBlock() Block {
	var block Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], sb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], sb.Next)
	for i, desc := range sb.Descriptors {
		copyBytes(block[0x04+i*0x27:0x04+(i+1)*0x27], desc.toBytes())
	}
	return block
}

// FromBlock unmarshals a Block of bytes into a SubdirectoryBlock.
func (sb *SubdirectoryBlock) FromBlock(block Block) error {
	sb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	sb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	for i := range sb.Descriptors {
		if err := sb.Descriptors[i].fromBytes(block[0x4+i*0x27 : 0x4+(i+1)*0x27]); err != nil {
			return fmt.Errorf("cannot deserialize file descriptor %d of Volume Directory Block: %v", err)
		}
	}
	return nil
}

type SubdirectoryHeader struct {
	TypeAndNameLength byte     // Storage type (top four bits) and subdirectory name length (lower four).
	SubdirectoryName  [15]byte // Subdirectory name (actual length defined in TypeAndNameLength)
	SeventyFive       byte     // Must contain $75 (!?)
	Unused1           [7]byte
	Creation          DateTime // Date and time volume was formatted
	Version           byte
	MinVersion        byte
	Access            Access
	EntryLength       byte   // Length of each entry in the Subdirectory: usually $27
	EntriesPerBlock   byte   // Usually $0D
	FileCount         uint16 // Number of active entries in the Subdirectory, not counting the Subdirectory Header
	ParentPointer     uint16 // The block number of the key (first) block of the directory that contains the entry that describes this subdirectory
	ParentEntry       byte   // Index in the parent directory for this subdirectory's entry (counting from parent header = 0)
	ParentEntryLength byte   // Usually $27
}

// toBytes converts a SubdirectoryHeader to a slice of bytes.
func (sh SubdirectoryHeader) toBytes() []byte {
	buf := make([]byte, 0x27)
	buf[0] = sh.TypeAndNameLength
	copyBytes(buf[1:0x10], sh.SubdirectoryName[:])
	buf[0x10] = sh.SeventyFive
	copyBytes(buf[0x11:0x18], sh.Unused1[:])
	copyBytes(buf[0x18:0x1c], sh.Creation.toBytes())
	buf[0x1c] = sh.Version
	buf[0x1d] = sh.MinVersion
	buf[0x1e] = byte(sh.Access)
	buf[0x1f] = sh.EntryLength
	buf[0x20] = sh.EntriesPerBlock
	binary.LittleEndian.PutUint16(buf[0x20:0x22], sh.FileCount)
	binary.LittleEndian.PutUint16(buf[0x22:0x24], sh.ParentPointer)
	buf[0x24] = sh.ParentEntry
	buf[0x25] = sh.ParentEntryLength
	return buf
}

// fromBytes unmarshals a slice of bytes into a SubdirectoryHeader.
func (sh *SubdirectoryHeader) fromBytes(buf []byte) error {
	if len(buf) != 0x27 {
		return fmt.Errorf("VolumeDirectoryHeader should be 0x27 bytes long; got 0x%02x", len(buf))
	}
	sh.TypeAndNameLength = buf[0]
	copyBytes(sh.SubdirectoryName[:], buf[1:0x10])
	if buf[0x10] != 0x75 {
		return fmt.Errorf("the byte after subdirectory name should be 0x75; got 0x%02x", buf[0x10])
	}
	sh.SeventyFive = buf[0x10]
	copyBytes(sh.Unused1[:], buf[0x11:0x18])
	if err := sh.Creation.fromBytes(buf[0x18:0x1c]); err != nil {
		return fmt.Errorf("unable to deserialize Subdirectory Header Creation date/time: %v", err)
	}
	sh.Version = buf[0x1c]
	sh.MinVersion = buf[0x1d]
	sh.Access = Access(buf[0x1e])
	sh.EntryLength = buf[0x1f]
	sh.EntriesPerBlock = buf[0x20]
	sh.FileCount = binary.LittleEndian.Uint16(buf[0x20:0x22])
	sh.ParentPointer = binary.LittleEndian.Uint16(buf[0x22:0x24])
	sh.ParentEntry = buf[0x24]
	sh.ParentEntryLength = buf[0x25]
	return nil
}

// copyBytes is just like the builtin copy, but just for byte slices,
// and it checks that dst and src have the same length.
func copyBytes(dst, src []byte) int {
	if len(dst) != len(src) {
		panic(fmt.Sprintf("copyBytes called with differing lengths %d and %d", len(dst), len(src)))
	}
	return copy(dst, src)
}
