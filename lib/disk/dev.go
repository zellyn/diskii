// Copyright © 2017 Zellyn Hunter <zellyn@gmail.com>

// dev.go contains logic for reading ".po" disk images.

package disk

import (
	"fmt"
	"io"
	"io/ioutil"
)

// Dev represents a .po disk image.
type Dev struct {
	data   []byte // The actual data in the file
	blocks uint16 // Number of blocks
}

var _ BlockDevice = (*Dev)(nil)

// LoadDev loads a .po image from a file.
func LoadDev(filename string) (Dev, error) {
	bb, err := ioutil.ReadFile(filename)
	if err != nil {
		return Dev{}, err
	}
	if len(bb)%512 != 0 {
		return Dev{}, fmt.Errorf("expected file %q to contain a multiple of 512 bytes, but got %d", filename, len(bb))
	}
	return Dev{
		data:   bb,
		blocks: uint16(len(bb) / 512),
	}, nil
}

// Empty creates a .po image that is all zeros.
func EmptyDev(blocks uint16) Dev {
	return Dev{
		data:   make([]byte, 512*int(blocks)),
		blocks: blocks,
	}
}

// ReadBlock reads a single block from the device. It always returns
// 512 byes.
func (d Dev) ReadBlock(index uint16) (Block, error) {
	var b Block
	copy(b[:], d.data[int(index)*512:int(index+1)*512])
	return b, nil
}

// WriteBlock writes a single block to a device. It expects exactly
// 512 bytes.
func (d Dev) WriteBlock(index uint16, data Block) error {
	copy(d.data[int(index)*512:int(index+1)*512], data[:])
	return nil
}

// Blocks returns the number of blocks in the device.
func (d Dev) Blocks() uint16 {
	return d.blocks
}

// Order returns the order of blocks on the device.
func (d Dev) Order() string {
	return "prodos"
}

// Write writes the device contents to the given file.
func (d Dev) Write(w io.Writer) (n int, err error) {
	return w.Write(d.data)
}
