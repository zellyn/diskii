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
const TrackLength = 6656

type Woz struct {
	Info     Info
	Unknowns []UnknownChunk
	TMap     [160]uint8
	TRKS     []TRK
	Metadata Metadata
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

type TRK struct {
	BitStream      [6646]uint8
	BytesUsed      uint16
	BitCount       uint16
	SplicePoint    uint16
	SpliceNibble   uint8
	SpliceBitCount uint8
	Reserved       uint16
}

type Metadata struct {
	Keys      []string
	RawValues map[string]string
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

	d.woz.Info.Version = d.tmp[0]
	d.woz.Info.DiskType = DiskType(d.tmp[1])
	d.woz.Info.WriteProtected = d.tmp[2] == 1
	d.woz.Info.Synchronized = d.tmp[3] == 1
	d.woz.Info.Cleaned = d.tmp[4] == 1
	d.woz.Info.Creator = strings.TrimRight(string(d.tmp[5:37]), " ")

	return nil
}

func (d *decoder) parseTMAP(length uint32) error {
	d.info("TMAP chunk!\n")
	if length != 160 {
		d.warn("expected TMAP chunk length of 160; got %d", length)
	}
	if _, err := io.ReadFull(d.r, d.woz.TMap[:]); err != nil {
		return err
	}
	d.crc.Write(d.woz.TMap[:])
	return nil
}

func (d *decoder) parseTRKS(length uint32) error {
	d.info("TRKS chunk!\n")
	if length%TrackLength != 0 {
		return FormatError(fmt.Sprintf("expected TRKS chunk length to be a multiple of %d; got %d", TrackLength, length))
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)

	for offset := 0; offset < int(length); offset += TrackLength {
		b := buf[offset : offset+TrackLength]
		t := TRK{
			BytesUsed:      binary.LittleEndian.Uint16(b[6646:6648]),
			BitCount:       binary.LittleEndian.Uint16(b[6648:6650]),
			SplicePoint:    binary.LittleEndian.Uint16(b[6650:6652]),
			SpliceNibble:   b[6652],
			SpliceBitCount: b[6653],
			Reserved:       binary.LittleEndian.Uint16(b[6654:6656]),
		}
		copy(t.BitStream[:], b)
		d.woz.TRKS = append(d.woz.TRKS, t)
	}

	// type TRK struct {
	// 	Bitstream      [6646]uint8
	// 	BytesUsed      uint16
	// 	BitCount       uint16
	// 	SplicePoint    uint16
	// 	SpliceNibble   uint8
	// 	SpliceBitCount uint8
	// 	Reserved       uint16
	// }

	return nil
}

func (d *decoder) parseMETA(length uint32) error {
	d.info("META chunk!\n")
	buf := make([]byte, length)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return err
	}
	d.crc.Write(buf)
	rows := strings.Split(string(buf), "\n")
	m := &d.woz.Metadata
	m.RawValues = make(map[string]string, len(rows))
	for _, row := range rows {
		parts := strings.SplitN(row, "\t", 2)
		if len(parts) == 0 {
			return FormatError("empty metadata line")
		}
		if len(parts) == 1 {
			return FormatError("strange metadata line with no tab: " + parts[0])
		}
		m.Keys = append(m.Keys, parts[0])
		m.RawValues[parts[0]] = parts[1]
	}

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
