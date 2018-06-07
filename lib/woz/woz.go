package woz

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"strings"
)

const wozHeader = "WOZ1\xFF\n\r\n"

type Woz struct {
	Info     Info
	Unknowns []UnknownChunk
}

type UnknownChunk struct {
	Id   string
	Data []byte
}

type DiskType uint8

const (
	DiskType525 DiskType = 1
	DiskType35  DiskType = 2
)

type Info struct {
	Version        uint8
	DiskType       DiskType
	WriteProtected bool
	Synchronized   bool
	Cleaned        bool
	Creator        string
}

type decoder struct {
	r      io.Reader
	woz    *Woz
	crc    hash.Hash32
	tmp    [3 * 256]byte
	crcVal uint32
}

// A FormatError reports that the input is not a valid woz file.
type FormatError string

func (e FormatError) Error() string { return "woz: invalid format: " + string(e) }

type CRCError struct {
	Declared uint32
	Computed uint32
}

func (e CRCError) Error() string {
	return fmt.Sprintf("woz: failed checksum: declared=%d; computed=%d", e.Declared, e.Computed)
}

func (d *decoder) info(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Printf("INFO: "+format, args...)
}

func (d *decoder) warn(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Printf("WARN: "+format, args...)
}

func (d *decoder) checkHeader() error {
	_, err := io.ReadFull(d.r, d.tmp[:len(wozHeader)])
	if err != nil {
		return err
	}
	if string(d.tmp[:len(wozHeader)]) != wozHeader {
		return FormatError("not a woz file")
	}
	if err := binary.Read(d.r, binary.LittleEndian, &d.crcVal); err != nil {
		return err
	}
	return nil
}

func (d *decoder) parseChunk() (done bool, err error) {
	// Read the chunk type and length
	n, err := io.ReadFull(d.r, d.tmp[:8])
	if err != nil {
		if n == 0 && err == io.EOF {
			return true, nil
		}
		return false, err
	}
	length := binary.LittleEndian.Uint32(d.tmp[4:8])
	d.crc.Write(d.tmp[:8])
	switch string(d.tmp[:4]) {
	case "INFO":
		return false, d.parseINFO(length)
	case "TMAP":
		return false, d.parseTMAP(length)
	case "TRKS":
		return false, d.parseTRKS(length)
	case "META":
		return false, d.parseMETA(length)
	default:
		return false, d.parseUnknown(string(d.tmp[:4]), length)
	}

	return false, nil
}

func (d *decoder) parseINFO(length uint32) error {
	d.info("INFO chunk!\n")
	if length != 60 {
		d.warn("expected INFO chunk length of 60; got %d", length)
	}
	if _, err := io.ReadFull(d.r, d.tmp[:length]); err != nil {
		return err
	}
	d.crc.Write(d.tmp[:length])
	return nil
}

func (d *decoder) parseTMAP(length uint32) error {
	d.info("TMAP chunk!\n")
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)
	return nil
}

func (d *decoder) parseTRKS(length uint32) error {
	d.info("TRKS chunk!\n")
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)
	return nil
}

func (d *decoder) parseMETA(length uint32) error {
	d.info("META chunk!\n")
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)
	return nil
}

func (d *decoder) parseUnknown(id string, length uint32) error {
	d.info("unknown chunk type (%s): ignoring\n", id)
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)
	d.woz.Unknowns = append(d.woz.Unknowns, UnknownChunk{Id: id, Data: buf})
	return nil
}

// Decode reads a woz disk image from r and returns it as a *Woz.
func Decode(r io.Reader) (*Woz, error) {
	d := &decoder{
		r:   r,
		crc: crc32.NewIEEE(),
		woz: &Woz{},
	}
	if err := d.checkHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	// Read all chunks.
	for {
		done, err := d.parseChunk()
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}

	// Check CRC.
	if d.crcVal != d.crc.Sum32() {
		return d.woz, CRCError{Declared: d.crcVal, Computed: d.crc.Sum32()}
	}

	return d.woz, nil
}
