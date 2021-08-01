// Copyright © 2017 Zellyn Hunter <zellyn@gmail.com>

// Package prodos contains routines for working with the on-device
// structures of Apple ProDOS.
package prodos

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/zellyn/diskii/disk"
	"github.com/zellyn/diskii/types"
)

// Storage types.
const (
	TypeDeleted               = 0
	TypeSeedling              = 0x1
	TypeSapling               = 0x2
	TypeTree                  = 0x3
	TypeSubdirectory          = 0xD
	TypeSubdirectoryHeader    = 0xE
	TypeVolumeDirectoryHeader = 0xF
)

// blockBase represents a 512-byte block of data.
type blockBase struct {
	block uint16 // Block index this data was loaded from.
}

// GetBlock gets the block index from a blockBase.
func (bb blockBase) GetBlock() uint16 {
	return bb.block
}

// SetBlock sets the block index of a blockBase.
func (bb *blockBase) SetBlock(block uint16) {
	bb.block = block
}

// A bitmapPart is a single block of a volumeBitMap.
type bitmapPart struct {
	blockBase
	data disk.Block
}

// Ensure that bitmapPart is valid BlockSource and BlockSink.
var _ disk.BlockSource = (*bitmapPart)(nil)
var _ disk.BlockSink = (*bitmapPart)(nil)

// FromBlock unmarshals a bitmapPart from a Block.
func (bp *bitmapPart) FromBlock(block disk.Block) error {
	bp.data = block
	return nil
}

// ToBlock marshals a bitmapPart struct to a block.
func (bp bitmapPart) ToBlock() (disk.Block, error) {
	return bp.data, nil
}

type VolumeBitMap []bitmapPart

func NewVolumeBitMap(startBlock uint16, blocks uint16) VolumeBitMap {
	vbm := VolumeBitMap(make([]bitmapPart, (blocks+(512*8)-1)/(512*8)))
	for i := range vbm {
		vbm[i].SetBlock(startBlock + uint16(i))
	}
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
		vbm[blockIndex].data[blockByteIndex] |= bit
	} else {
		vbm[blockIndex].data[blockByteIndex] &^= bit
	}
}

// IsFree returns true if the given block on the device is free,
// according to the VolumeBitMap.
func (vbm VolumeBitMap) IsFree(block uint16) bool {
	byteIndex := block >> 3
	blockIndex := byteIndex / 512
	blockByteIndex := byteIndex % 512
	bit := byte(1 << (7 - (block & 7)))
	return vbm[blockIndex].data[blockByteIndex]&bit > 0
}

// readVolumeBitMap reads the entire volume bitmap from a block
// device.
func readVolumeBitMap(devicebytes []byte, startBlock uint16) (VolumeBitMap, error) {
	blocks := uint16(len(devicebytes) / 512 / 4096)
	vbm := NewVolumeBitMap(startBlock, blocks)
	for i := 0; i < len(vbm); i++ {
		if err := disk.UnmarshalBlock(devicebytes, &vbm[i], vbm[i].GetBlock()); err != nil {
			return nil, fmt.Errorf("cannot read block %d (device block %d) of Volume Bit Map: %v", i, vbm[i].GetBlock(), err)
		}
	}
	return vbm, nil
}

// Write writes the Volume Bit Map to a block device.
func (vbm VolumeBitMap) Write(devicebytes []byte) error {
	for i, bp := range vbm {
		if err := disk.MarshalBlock(devicebytes, bp); err != nil {
			return fmt.Errorf("cannot write block %d (device block %d) of Volume Bit Map: %v", i, bp.GetBlock(), err)
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
func (dt *DateTime) fromBytes(b []byte) {
	if len(b) != 4 {
		panic(fmt.Sprintf("DateTime expects 4 bytes; got %d", len(b)))
	}
	dt.YMD[0] = b[0]
	dt.YMD[1] = b[1]
	dt.HM[0] = b[2]
	dt.HM[1] = b[3]
}

// Validate checks a DateTime for problems, returning a slice of errors.
func (dt DateTime) Validate(fieldDescription string) (errors []error) {
	if dt.HM[0] >= 24 {
		errors = append(errors, fmt.Errorf("%s expects hour<24; got %d", fieldDescription, dt.HM[0]))
	}
	if dt.HM[1] >= 60 {
		errors = append(errors, fmt.Errorf("%s expects minute<60; got %x", fieldDescription, dt.HM[1]))
	}
	return errors
}

// VolumeDirectoryKeyBlock is the struct used to hold the ProDOS Volume Directory Key
// Block structure.  See page 4-4 of Beneath Apple ProDOS.
type VolumeDirectoryKeyBlock struct {
	blockBase
	Prev        uint16 // Pointer to previous block (always zero: the KeyBlock is the first Volume Directory block
	Next        uint16 // Pointer to next block in the Volume Directory
	Header      VolumeDirectoryHeader
	Descriptors [12]FileDescriptor
	Extra       byte // Trailing byte (so we don't lose it)
}

// Ensure that VolumeDirectoryKeyBlock is valid BlockSource and BlockSink.
var _ disk.BlockSource = (*VolumeDirectoryKeyBlock)(nil)
var _ disk.BlockSink = (*VolumeDirectoryKeyBlock)(nil)

// ToBlock marshals the VolumeDirectoryKeyBlock to a Block of bytes.
func (vdkb VolumeDirectoryKeyBlock) ToBlock() (disk.Block, error) {
	var block disk.Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], vdkb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], vdkb.Next)
	copyBytes(block[0x04:0x02b], vdkb.Header.toBytes())
	for i, desc := range vdkb.Descriptors {
		copyBytes(block[0x2b+i*0x27:0x2b+(i+1)*0x27], desc.toBytes())
	}
	block[511] = vdkb.Extra
	return block, nil
}

// FromBlock unmarshals a Block of bytes into a VolumeDirectoryKeyBlock.
func (vdkb *VolumeDirectoryKeyBlock) FromBlock(block disk.Block) error {
	vdkb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	vdkb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	vdkb.Header.fromBytes(block[0x04:0x2b])
	for i := range vdkb.Descriptors {
		vdkb.Descriptors[i].fromBytes(block[0x2b+i*0x27 : 0x2b+(i+1)*0x27])
	}
	vdkb.Extra = block[511]
	return nil
}

// Validate validates a VolumeDirectoryKeyBlock for valid values.
func (vdkb VolumeDirectoryKeyBlock) Validate() (errors []error) {
	if vdkb.Prev != 0 {
		errors = append(errors, fmt.Errorf("Volume Directory Key Block should have a `Previous` block of 0, got $%04x", vdkb.Prev))
	}
	errors = append(errors, vdkb.Header.Validate()...)
	for _, desc := range vdkb.Descriptors {
		errors = append(errors, desc.Validate()...)
	}
	if vdkb.Extra != 0 {
		errors = append(errors, fmt.Errorf("expected last byte of Volume Directory Key Block == 0x0; got 0x%02x", vdkb.Extra))
	}
	return errors
}

// VolumeDirectoryBlock is a normal (non-key) segment in the Volume Directory Header.
type VolumeDirectoryBlock struct {
	blockBase
	Prev        uint16 // Pointer to previous block in the Volume Directory.
	Next        uint16 // Pointer to next block in the Volume Directory.
	Descriptors [13]FileDescriptor
	Extra       byte // Trailing byte (so we don't lose it)
}

// Ensure that VolumeDirectoryBlock is valid BlockSource and BlockSink.
var _ disk.BlockSource = (*VolumeDirectoryBlock)(nil)
var _ disk.BlockSink = (*VolumeDirectoryBlock)(nil)

// ToBlock marshals a VolumeDirectoryBlock to a Block of bytes.
func (vdb VolumeDirectoryBlock) ToBlock() (disk.Block, error) {
	var block disk.Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], vdb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], vdb.Next)
	for i, desc := range vdb.Descriptors {
		copyBytes(block[0x04+i*0x27:0x04+(i+1)*0x27], desc.toBytes())
	}
	block[511] = vdb.Extra
	return block, nil
}

// FromBlock unmarshals a Block of bytes into a VolumeDirectoryBlock.
func (vdb *VolumeDirectoryBlock) FromBlock(block disk.Block) error {
	vdb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	vdb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	for i := range vdb.Descriptors {
		vdb.Descriptors[i].fromBytes(block[0x4+i*0x27 : 0x4+(i+1)*0x27])
	}
	vdb.Extra = block[511]
	return nil
}

// Validate validates a VolumeDirectoryBlock for valid values.
func (vdb VolumeDirectoryBlock) Validate() (errors []error) {
	for _, desc := range vdb.Descriptors {
		errors = append(errors, desc.Validate()...)
	}
	if vdb.Extra != 0 {
		errors = append(errors, fmt.Errorf("expected last byte of Volume Directory Block == 0x0; got 0x%02x", vdb.Extra))
	}
	return errors
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
	binary.LittleEndian.PutUint16(buf[0x21:0x23], vdh.FileCount)
	binary.LittleEndian.PutUint16(buf[0x23:0x25], vdh.BitMapPointer)
	binary.LittleEndian.PutUint16(buf[0x25:0x27], vdh.TotalBlocks)
	return buf
}

// fromBytes unmarshals a slice of bytes into a VolumeDirectoryHeader.
func (vdh *VolumeDirectoryHeader) fromBytes(buf []byte) {
	if len(buf) != 0x27 {
		panic(fmt.Sprintf("VolumeDirectoryHeader should be 0x27 bytes long; got 0x%02x", len(buf)))
	}
	vdh.TypeAndNameLength = buf[0]
	copyBytes(vdh.VolumeName[:], buf[1:0x10])
	copyBytes(vdh.Unused1[:], buf[0x10:0x18])
	vdh.Creation.fromBytes(buf[0x18:0x1c])
	vdh.Version = buf[0x1c]
	vdh.MinVersion = buf[0x1d]
	vdh.Access = Access(buf[0x1e])
	vdh.EntryLength = buf[0x1f]
	vdh.EntriesPerBlock = buf[0x20]
	vdh.FileCount = binary.LittleEndian.Uint16(buf[0x21:0x23])
	vdh.BitMapPointer = binary.LittleEndian.Uint16(buf[0x23:0x25])
	vdh.TotalBlocks = binary.LittleEndian.Uint16(buf[0x25:0x27])
}

// Validate validates a VolumeDirectoryHeader for valid values.
func (vdh VolumeDirectoryHeader) Validate() (errors []error) {
	errors = append(errors, vdh.Creation.Validate("creation date/time of VolumeDirectoryHeader")...)
	return errors
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
	EOF               [3]byte  // 3-byte offset of EOF from first byte. For sequential files, just the length
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

// descriptor returns a types.Descriptor for a FileDescriptor.
func (fd FileDescriptor) descriptor() types.Descriptor {
	desc := types.Descriptor{
		Name:   fd.Name(),
		Blocks: int(fd.BlocksUsed),
		Length: int(fd.EOF[0]) + int(fd.EOF[1])<<8 + int(fd.EOF[2])<<16,
		Locked: false, // TODO(zellyn): use prodos-style access in types.Descriptor
		Type:   types.Filetype(fd.FileType),
	}
	return desc
}

// Name returns the string filename of a file descriptor.
func (fd FileDescriptor) Name() string {
	return string(fd.FileName[0 : fd.TypeAndNameLength&0xf])
}

// Type returns the type of a file descriptor.
func (fd FileDescriptor) Type() byte {
	return fd.TypeAndNameLength >> 4
}

// toBytes converts a FileDescriptor to a slice of bytes.
func (fd FileDescriptor) toBytes() []byte {
	buf := make([]byte, 0x27)
	buf[0] = fd.TypeAndNameLength
	copyBytes(buf[1:0x10], fd.FileName[:])
	buf[0x10] = fd.FileType
	binary.LittleEndian.PutUint16(buf[0x11:0x13], fd.KeyPointer)
	binary.LittleEndian.PutUint16(buf[0x13:0x15], fd.BlocksUsed)
	copyBytes(buf[0x15:0x18], fd.EOF[:])
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
func (fd *FileDescriptor) fromBytes(buf []byte) {
	if len(buf) != 0x27 {
		panic(fmt.Sprintf("FileDescriptor should be 0x27 bytes long; got 0x%02x", len(buf)))
	}
	fd.TypeAndNameLength = buf[0]
	copyBytes(fd.FileName[:], buf[1:0x10])
	fd.FileType = buf[0x10]
	fd.KeyPointer = binary.LittleEndian.Uint16(buf[0x11:0x13])
	fd.BlocksUsed = binary.LittleEndian.Uint16(buf[0x13:0x15])
	copyBytes(fd.EOF[:], buf[0x15:0x18])
	fd.Creation.fromBytes(buf[0x18:0x1c])
	fd.Version = buf[0x1c]
	fd.MinVersion = buf[0x1d]
	fd.Access = Access(buf[0x1e])
	fd.AuxType = binary.LittleEndian.Uint16(buf[0x1f:0x21])
	fd.LastMod.fromBytes(buf[0x21:0x25])
	fd.HeaderPointer = binary.LittleEndian.Uint16(buf[0x25:0x27])
}

// Validate validates a FileDescriptor for valid values.
func (fd FileDescriptor) Validate() (errors []error) {
	errors = append(errors, fd.Creation.Validate(fmt.Sprintf("creation date/time of FileDescriptor %q", fd.Name()))...)
	errors = append(errors, fd.LastMod.Validate(fmt.Sprintf("last modification date/time of FileDescriptor %q", fd.Name()))...)
	return errors
}

// An index block contains 256 16-bit block numbers, pointing to other
// blocks. The LSBs are stored in the first half, MSBs in the second.
type IndexBlock disk.Block

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
	blockBase
	Prev        uint16 // Pointer to previous block (always zero: the KeyBlock is the first Volume Directory block
	Next        uint16 // Pointer to next block in the Volume Directory
	Header      SubdirectoryHeader
	Descriptors [12]FileDescriptor
	Extra       byte // Trailing byte (so we don't lose it)
}

// Ensure that SubdirectoryKeyBlock is valid BlockSource and BlockSink.
var _ disk.BlockSource = (*SubdirectoryKeyBlock)(nil)
var _ disk.BlockSink = (*SubdirectoryKeyBlock)(nil)

// ToBlock marshals the SubdirectoryKeyBlock to a Block of bytes.
func (skb SubdirectoryKeyBlock) ToBlock() (disk.Block, error) {
	var block disk.Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], skb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], skb.Next)
	copyBytes(block[0x04:0x02b], skb.Header.toBytes())
	for i, desc := range skb.Descriptors {
		copyBytes(block[0x2b+i*0x27:0x2b+(i+1)*0x27], desc.toBytes())
	}
	block[511] = skb.Extra
	return block, nil
}

// FromBlock unmarshals a Block of bytes into a SubdirectoryKeyBlock.
func (skb *SubdirectoryKeyBlock) FromBlock(block disk.Block) error {
	skb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	skb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	skb.Header.fromBytes(block[0x04:0x2b])
	for i := range skb.Descriptors {
		skb.Descriptors[i].fromBytes(block[0x2b+i*0x27 : 0x2b+(i+1)*0x27])
	}
	skb.Extra = block[511]
	return nil
}

// Validate validates a SubdirectoryKeyBlock for valid values.
func (skb SubdirectoryKeyBlock) Validate() (errors []error) {
	if skb.Prev != 0 {
		errors = append(errors, fmt.Errorf("Subdirectory Key Block should have a `Previous` block of 0, got $%04x", skb.Prev))
	}
	errors = append(errors, skb.Header.Validate()...)
	for _, desc := range skb.Descriptors {
		errors = append(errors, desc.Validate()...)
	}
	if skb.Extra != 0 {
		errors = append(errors, fmt.Errorf("expected last byte of Subdirectory Key Block == 0x0; got 0x%02x", skb.Extra))
	}
	return errors
}

// SubdirectoryBlock is a normal (non-key) segment in a Subdirectory.
type SubdirectoryBlock struct {
	blockBase
	Prev        uint16 // Pointer to previous block in the Volume Directory.
	Next        uint16 // Pointer to next block in the Volume Directory.
	Descriptors [13]FileDescriptor
	Extra       byte // Trailing byte (so we don't lose it)
}

// Ensure that SubdirectoryBlock is valid BlockSource and BlockSink.
var _ disk.BlockSource = (*SubdirectoryBlock)(nil)
var _ disk.BlockSink = (*SubdirectoryBlock)(nil)

// ToBlock marshals a SubdirectoryBlock to a Block of bytes.
func (sb SubdirectoryBlock) ToBlock() (disk.Block, error) {
	var block disk.Block
	binary.LittleEndian.PutUint16(block[0x0:0x2], sb.Prev)
	binary.LittleEndian.PutUint16(block[0x2:0x4], sb.Next)
	for i, desc := range sb.Descriptors {
		copyBytes(block[0x04+i*0x27:0x04+(i+1)*0x27], desc.toBytes())
	}
	block[511] = sb.Extra
	return block, nil
}

// FromBlock unmarshals a Block of bytes into a SubdirectoryBlock.
func (sb *SubdirectoryBlock) FromBlock(block disk.Block) error {
	sb.Prev = binary.LittleEndian.Uint16(block[0x0:0x2])
	sb.Next = binary.LittleEndian.Uint16(block[0x2:0x4])
	for i := range sb.Descriptors {
		sb.Descriptors[i].fromBytes(block[0x4+i*0x27 : 0x4+(i+1)*0x27])
	}
	sb.Extra = block[511]
	return nil
}

// Validate validates a SubdirectoryBlock for valid values.
func (sb SubdirectoryBlock) Validate() (errors []error) {
	for _, desc := range sb.Descriptors {
		errors = append(errors, desc.Validate()...)
	}
	if sb.Extra != 0 {
		errors = append(errors, fmt.Errorf("expected last byte of Subdirectory Block == 0x0; got 0x%02x", sb.Extra))
	}
	return errors
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
	binary.LittleEndian.PutUint16(buf[0x21:0x23], sh.FileCount)
	binary.LittleEndian.PutUint16(buf[0x23:0x25], sh.ParentPointer)
	buf[0x25] = sh.ParentEntry
	buf[0x26] = sh.ParentEntryLength
	return buf
}

// fromBytes unmarshals a slice of bytes into a SubdirectoryHeader.
func (sh *SubdirectoryHeader) fromBytes(buf []byte) {
	if len(buf) != 0x27 {
		panic(fmt.Sprintf("VolumeDirectoryHeader should be 0x27 bytes long; got 0x%02x", len(buf)))
	}
	sh.TypeAndNameLength = buf[0]
	copyBytes(sh.SubdirectoryName[:], buf[1:0x10])
	sh.SeventyFive = buf[0x10]
	copyBytes(sh.Unused1[:], buf[0x11:0x18])
	sh.Creation.fromBytes(buf[0x18:0x1c])
	sh.Version = buf[0x1c]
	sh.MinVersion = buf[0x1d]
	sh.Access = Access(buf[0x1e])
	sh.EntryLength = buf[0x1f]
	sh.EntriesPerBlock = buf[0x20]
	sh.FileCount = binary.LittleEndian.Uint16(buf[0x21:0x23])
	sh.ParentPointer = binary.LittleEndian.Uint16(buf[0x23:0x25])
	sh.ParentEntry = buf[0x25]
	sh.ParentEntryLength = buf[0x26]
}

// Validate validates a SubdirectoryHeader for valid values.
func (sh SubdirectoryHeader) Validate() (errors []error) {
	if sh.SeventyFive != 0x75 {
		errors = append(errors, fmt.Errorf("byte after subdirectory name %q should be 0x75; got 0x%02x", sh.Name(), sh.SeventyFive))
	}
	errors = append(errors, sh.Creation.Validate(fmt.Sprintf("subdirectory %q header creation date/time", sh.Name()))...)
	return errors
}

// Name returns the string filename of a subdirectory header.
func (sh SubdirectoryHeader) Name() string {
	return string(sh.SubdirectoryName[0 : sh.TypeAndNameLength&0xf])
}

// Volume is the in-memory representation of a device's volume
// information.
type Volume struct {
	keyBlock          *VolumeDirectoryKeyBlock // The key block describing the entire volume
	blocks            []*VolumeDirectoryBlock  // The blocks in the top-level volume
	bitmap            *VolumeBitMap            // Bitmap of which blocks are free
	subdirsByBlock    map[uint16]*Subdirectory // A mapping of block number to subdirectory object
	subdirsByName     map[string]*Subdirectory // a mapping of string to subdirectory object
	firstSubdirBlocks map[uint16]uint16        // A mapping of later dir/subdir blocks to the first one in the chain
}

// Subdirectory is the in-memory representation of a single
// subdirectory's information.
type Subdirectory struct {
	keyBlock *SubdirectoryKeyBlock
	blocks   []*SubdirectoryBlock
}

// descriptors returns a slice of all top-level file descriptors in a
// volume, deleted or not.
func (v Volume) descriptors() []FileDescriptor {
	var descs []FileDescriptor

	descs = append(descs, v.keyBlock.Descriptors[:]...)
	for _, block := range v.blocks {
		descs = append(descs, block.Descriptors[:]...)
	}
	return descs
}

// subdirDescriptors returns a slice of all top-level file descriptors
// in a volume that are subdirectories.
func (v Volume) subdirDescriptors() []FileDescriptor {
	var descs []FileDescriptor

	for _, desc := range v.descriptors() {
		if desc.Type() == TypeSubdirectory {
			descs = append(descs, desc)
		}
	}
	return descs
}

// readVolume reads the entire volume and subdirectories from a device
// into memory.
func readVolume(devicebytes []byte, keyBlock uint16, debug bool) (Volume, error) {
	v := Volume{
		keyBlock:          &VolumeDirectoryKeyBlock{},
		subdirsByBlock:    make(map[uint16]*Subdirectory),
		subdirsByName:     make(map[string]*Subdirectory),
		firstSubdirBlocks: make(map[uint16]uint16),
	}

	if err := disk.UnmarshalBlock(devicebytes, v.keyBlock, keyBlock); err != nil {
		return v, fmt.Errorf("cannot read first block of volume directory (block %d): %v", keyBlock, err)
	}
	// if debug {
	// 	fmt.Fprintf(os.Stderr, "keyblock: %#v\n", v.keyBlock)
	// }

	vbm, err := readVolumeBitMap(devicebytes, v.keyBlock.Header.BitMapPointer)
	if err != nil {
		return v, err
	}

	v.bitmap = &vbm

	// if debug {
	// 	fmt.Fprintf(os.Stderr, "volume bitmap: %#v\n", v.bitmap)
	// }

	for block := v.keyBlock.Next; block != 0; block = v.blocks[len(v.blocks)-1].Next {
		vdb := VolumeDirectoryBlock{}
		if err := disk.UnmarshalBlock(devicebytes, &vdb, block); err != nil {
			return v, err
		}
		v.blocks = append(v.blocks, &vdb)
		v.firstSubdirBlocks[block] = keyBlock
		if debug {
			fmt.Fprintf(os.Stderr, "  firstSubdirBlocks[%d] → %d\n", block, keyBlock)
		}
		// if debug {
		// 	fmt.Fprintf(os.Stderr, "block: %#v\n", vdb)
		// }
	}

	sdds := v.subdirDescriptors()
	if debug {
		fmt.Fprintf(os.Stderr, "got %d top-level subdir descriptors\n", len(sdds))
	}

	for i := 0; i < len(sdds); i++ {
		sdd := sdds[i]
		sub, err := readSubdirectory(devicebytes, sdd)
		if err != nil {
			return v, err
		}
		v.subdirsByBlock[sdd.KeyPointer] = &sub
		if debug {
			fmt.Fprintf(os.Stderr, " subdirsByBlock[%d] → %q\n", sdd.KeyPointer, sub.keyBlock.Header.Name())
		}
		sdds = append(sdds, sub.subdirDescriptors()...)
		for _, block := range sub.blocks {
			v.firstSubdirBlocks[block.block] = sdd.KeyPointer
			if debug {
				fmt.Fprintf(os.Stderr, "  firstSubdirBlocks[%d] → %d\n", block.block, sdd.KeyPointer)
			}
		}
	}
	if debug {
		fmt.Fprintf(os.Stderr, "got %d total subdir descriptors\n", len(sdds))
	}

	for _, sd := range v.subdirsByBlock {
		name := sd.keyBlock.Header.Name()
		if debug {
			fmt.Fprintf(os.Stderr, "processing subdir %q\n", name)
		}
		parentName, err := parentDirName(sd.keyBlock.Header.ParentPointer, keyBlock, v.subdirsByBlock, v.firstSubdirBlocks)
		if err != nil {
			return v, err
		}
		if parentName != "" {
			name = parentName + "/" + name
		}

		v.subdirsByName[name] = sd
	}
	if debug {
		fmt.Fprintf(os.Stderr, "HERE2\n")
	}
	return v, nil
}

// descriptors returns a slice of all top-level file descriptors in a
// subdirectory, deleted or not.
func (s Subdirectory) descriptors() []FileDescriptor {
	var descs []FileDescriptor

	descs = append(descs, s.keyBlock.Descriptors[:]...)
	for _, block := range s.blocks {
		descs = append(descs, block.Descriptors[:]...)
	}
	return descs
}

// subdirDescriptors returns a slice of all top-level file descriptors
// in a subdirectory that are subdirectories.
func (s Subdirectory) subdirDescriptors() []FileDescriptor {
	var descs []FileDescriptor

	for _, desc := range s.descriptors() {
		if desc.Type() == TypeSubdirectory {
			descs = append(descs, desc)
		}
	}
	return descs
}

// parentDirName returns the full recursive directory name of the given parent directory.
func parentDirName(parentDirectoryBlock uint16, keyBlock uint16, subdirMap map[uint16]*Subdirectory, firstSubdirBlockMap map[uint16]uint16) (string, error) {
	if parentDirectoryBlock == keyBlock || firstSubdirBlockMap[parentDirectoryBlock] == keyBlock {
		return "", nil
	}
	sd := subdirMap[parentDirectoryBlock]
	if sd == nil {
		parentFirstBlock, ok := firstSubdirBlockMap[parentDirectoryBlock]
		if ok {
			sd = subdirMap[parentFirstBlock]
		}
	}
	if sd == nil {
		return "", fmt.Errorf("unable to find subdirectory for block %d", parentDirectoryBlock)
	}

	parentName, err := parentDirName(sd.keyBlock.Header.ParentPointer, keyBlock, subdirMap, firstSubdirBlockMap)
	if err != nil {
		return "", err
	}
	if parentName == "" {
		return sd.keyBlock.Header.Name(), nil
	}

	return parentName + "/" + sd.keyBlock.Header.Name(), nil
}

// readSubdirectory reads a single subdirectory from a device into
// memory.
func readSubdirectory(devicebytes []byte, fd FileDescriptor) (Subdirectory, error) {
	s := Subdirectory{
		keyBlock: &SubdirectoryKeyBlock{},
	}

	if err := disk.UnmarshalBlock(devicebytes, s.keyBlock, fd.KeyPointer); err != nil {
		return s, fmt.Errorf("cannot read first block of subdirectory %q (block %d): %v", fd.Name(), fd.KeyPointer, err)
	}

	for block := s.keyBlock.Next; block != 0; block = s.blocks[len(s.blocks)-1].Next {
		sdb := SubdirectoryBlock{}
		if err := disk.UnmarshalBlock(devicebytes, &sdb, block); err != nil {
			return s, err
		}
		s.blocks = append(s.blocks, &sdb)
	}

	return s, nil
}

// copyBytes is just like the builtin copy, but just for byte slices,
// and it checks that dst and src have the same length.
func copyBytes(dst, src []byte) int {
	if len(dst) != len(src) {
		panic(fmt.Sprintf("copyBytes called with differing lengths %d and %d", len(dst), len(src)))
	}
	return copy(dst, src)
}

// operator is a disk.Operator - an interface for performing
// high-level operations on files and directories.
type operator struct {
	data  []byte
	debug bool
}

var _ types.Operator = operator{}

// operatorName is the keyword name for the operator that undestands
// prodos disks/devices.
const operatorName = "prodos"

// Name returns the name of the operator.
func (o operator) Name() string {
	return operatorName
}

// HasSubdirs returns true if the underlying operating system on the
// disk allows subdirectories.
func (o operator) HasSubdirs() bool {
	return true
}

// Catalog returns a catalog of disk entries. subdir should be empty
// for operating systems that do not support subdirectories.
func (o operator) Catalog(subdir string) ([]types.Descriptor, error) {
	if o.debug {
		fmt.Fprintf(os.Stderr, "Catalog of %q\n", subdir)
	}
	vol, err := readVolume(o.data, 2, o.debug)
	if err != nil {
		return nil, fmt.Errorf("error reading volume: %w", err)
	}

	var result []types.Descriptor

	if subdir == "" {
		for _, desc := range vol.descriptors() {
			if desc.Type() != TypeDeleted {
				result = append(result, desc.descriptor())
			}
		}
		return result, nil
	}

	sd, ok := vol.subdirsByName[subdir]
	if !ok {
		return nil, fmt.Errorf("subdirectory %q not found", subdir)
	}

	for _, desc := range sd.descriptors() {
		if desc.Type() != TypeDeleted {
			result = append(result, desc.descriptor())
		}
	}
	return result, nil
}

// GetFile retrieves a file by name.
func (o operator) GetFile(filename string) (types.FileInfo, error) {
	return types.FileInfo{}, fmt.Errorf("%s doesn't implement GetFile yet", operatorName)
}

// Delete deletes a file by name. It returns true if the file was
// deleted, false if it didn't exist.
func (o operator) Delete(filename string) (bool, error) {
	return false, fmt.Errorf("%s doesn't implement Delete yet", operatorName)
}

// PutFile writes a file by name. If the file exists and overwrite
// is false, it returns with an error. Otherwise it returns true if
// an existing file was overwritten.
func (o operator) PutFile(fileInfo types.FileInfo, overwrite bool) (existed bool, err error) {
	return false, fmt.Errorf("%s doesn't implement PutFile yet", operatorName)
}

// DiskOrder returns the Physical-to-Logical mapping order.
func (o operator) DiskOrder() types.DiskOrder {
	return types.DiskOrderPO
}

// GetBytes returns the disk image bytes, in logical order.
func (o operator) GetBytes() []byte {
	return o.data
}

// OperatorFactory is a types.OperatorFactory for ProDos disks.
type OperatorFactory struct {
}

// Name returns the name of the operator.
func (of OperatorFactory) Name() string {
	return operatorName
}

// SeemsToMatch returns true if the []byte disk image seems to match the
// system of this operator.
func (of OperatorFactory) SeemsToMatch(devicebytes []byte, debug bool) bool {
	// For now, just return true if we can run Catalog successfully.
	_, err := readVolume(devicebytes, 2, debug)
	return err == nil
}

// Operator returns an Operator for the []byte disk image.
func (of OperatorFactory) Operator(devicebytes []byte, debug bool) (types.Operator, error) {
	return operator{data: devicebytes, debug: debug}, nil
}

// DiskOrder returns the Physical-to-Logical mapping order.
func (of OperatorFactory) DiskOrder() types.DiskOrder {
	return operator{}.DiskOrder()
}
