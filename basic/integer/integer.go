// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package integer provides routines for working with Integer BASIC
// files. References were
// http://fileformats.archiveteam.org/wiki/Apple_Integer_BASIC_tokenized_file
// and
// https://groups.google.com/d/msg/comp.sys.apple2/uwQbx1P94s4/Wk5YKuBhXRsJ
package integer

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// TokensByCode is a map from byte value to token text.
var TokensByCode = map[byte]string{
	0x00: "HIMEM:",
	0x01: "<token-01>",
	0x02: "_",
	0x03: ":",
	0x04: "LOAD",
	0x05: "SAVE",
	0x06: "CON",
	0x07: "RUN",
	0x08: "RUN",
	0x09: "DEL",
	0x0A: ",",
	0x0B: "NEW",
	0x0C: "CLR",
	0x0D: "AUTO",
	0x0E: ",",
	0x0F: "MAN",
	0x10: "HIMEM:",
	0x11: "LOMEM:",
	0x12: "+",
	0x13: "-",
	0x14: "*",
	0x15: "/",
	0x16: "=",
	0x17: "#",
	0x18: ">=",
	0x19: ">",
	0x1A: "<=",
	0x1B: "<>",
	0x1C: "<",
	0x1D: "AND",
	0x1E: "OR",
	0x1F: "MOD",
	0x20: "^",
	0x21: "+",
	0x22: "(",
	0x23: ",",
	0x24: "THEN",
	0x25: "THEN",
	0x26: ",",
	0x27: ",",
	0x28: `"`,
	0x29: `"`,
	0x2A: "(",
	0x2B: "!",
	0x2C: "!",
	0x2D: "(",
	0x2E: "PEEK",
	0x2F: "RND",
	0x30: "SGN",
	0x31: "ABS",
	0x32: "PDL",
	0x33: "RNDX",
	0x34: "(",
	0x35: "+",
	0x36: "-",
	0x37: "NOT",
	0x38: "(",
	0x39: "=",
	0x3A: "#",
	0x3B: "LEN(",
	0x3C: "ASC(",
	0x3D: "SCRN(",
	0x3E: ",",
	0x3F: "(",
	0x40: "$",
	0x41: "$",
	0x42: "(",
	0x43: ",",
	0x44: ",",
	0x45: ";",
	0x46: ";",
	0x47: ";",
	0x48: ",",
	0x49: ",",
	0x4A: ",",
	0x4B: "TEXT",
	0x4C: "GR",
	0x4D: "CALL",
	0x4E: "DIM",
	0x4F: "DIM",
	0x50: "TAB",
	0x51: "END",
	0x52: "INPUT",
	0x53: "INPUT",
	0x54: "INPUT",
	0x55: "FOR",
	0x56: "=",
	0x57: "TO",
	0x58: "STEP",
	0x59: "NEXT",
	0x5A: ",",
	0x5B: "RETURN",
	0x5C: "GOSUB",
	0x5D: "REM",
	0x5E: "LET",
	0x5F: "GOTO",
	0x60: "IF",
	0x61: "PRINT",
	0x62: "PRINT",
	0x63: "PRINT",
	0x64: "POKE",
	0x65: ",",
	0x66: "COLOR=",
	0x67: "PLOT",
	0x68: ",",
	0x69: "HLIN",
	0x6A: ",",
	0x6B: "AT",
	0x6C: "VLIN",
	0x6D: ",",
	0x6E: "AT",
	0x6F: "VTAB",
	0x70: "=",
	0x71: "=",
	0x72: ")",
	0x73: ")",
	0x74: "LIST",
	0x75: ",",
	0x76: "LIST",
	0x77: "POP",
	0x78: "NODSP",
	0x79: "DSP",
	0x7A: "NOTRACE",
	0x7B: "DSP",
	0x7C: "DSP",
	0x7D: "TRACE",
	0x7E: "PR#",
	0x7F: "IN#",
}

// Listing holds a listing of an entire BASIC program.
type Listing []Line

// Line holds a single BASIC line, with line number and text.
type Line struct {
	Num   int
	Bytes []byte
}

// Decode turns a raw binary file into a basic program.
func Decode(raw []byte) (Listing, error) {
	// First two bytes of Integer BASIC files on disk are length. Let's
	// be tolerant to getting either format.
	if len(raw) >= 2 {
		size := int(raw[0]) + (256 * int(raw[1]))
		if size == len(raw)-2 || size == len(raw)-3 {
			raw = raw[2:]
		}
	}

	listing := []Line{}
	for len(raw) > 3 {
		size := int(raw[0])
		num := int(raw[1]) + 256*int(raw[2])
		if len(raw) < size {
			return nil, fmt.Errorf("line %d wants %d bytes; only %d remain", num, size, len(raw))
		}
		if raw[size-1] != 1 {
			return nil, fmt.Errorf("line %d not terminated by 0x01", num)
		}
		listing = append(listing, Line{
			Num:   num,
			Bytes: raw[3 : size-1],
		})
		raw = raw[size:]
	}

	return listing, nil
}

/*
const (
	tokenREM        = 0x5D
	tokenUnaryPlus  = 0x35
	tokenUnaryMinus = 0x36
	tokenQuoteStart = 0x28
	tokenQuoteEnd   = 0x29
)
*/

func isalnum(b byte) bool {
	switch {
	case '0' <= b && b <= '9':
		return true
	case 'a' <= b && b <= 'z':
		return true
	case 'A' <= b && b <= 'Z':
		return true
	}
	return false
}

func (l Line) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%5d ", l.Num)
	var lastAN bool
	for i := 0; i < len(l.Bytes); i++ {
		ch := l.Bytes[i]
		if ch < 0x80 {
			lastAN = false
			token := TokensByCode[ch]
			buf.WriteString(token)
			if len(token) > 1 {
				buf.WriteByte(' ')
			}
			if token == "REM" {
				for _, ch := range l.Bytes[i+1:] {
					buf.WriteByte(ch - 0x80)
				}
				break
			}
		} else {
			ch = ch - 0x80
			if !lastAN && ch >= '0' && ch <= '9' {
				if len(l.Bytes) < i+3 {
					buf.WriteByte('?')
				} else {
					d := int(l.Bytes[i+1]) + 256*int(l.Bytes[i+2])
					i += 2
					buf.WriteString(strconv.Itoa(d))
				}
			} else {
				buf.WriteByte(ch)
			}
			lastAN = isalnum(ch)
		}
	}
	return buf.String()
}

func (l Listing) String() string {
	var buf bytes.Buffer
	for _, line := range l {
		buf.WriteString(line.String())
		buf.WriteByte('\n')
	}
	return buf.String()
}

var controlCharRegexp = regexp.MustCompile(`[\x00-\x1F]`)

// ChevronControlCodes converts ASCII control characters like chr(4)
// to chevron-surrounded codes like «ctrl-D».
func ChevronControlCodes(s string) string {
	return controlCharRegexp.ReplaceAllStringFunc(s, func(s string) string {
		if s == "\n" || s == "\t" {
			return s
		}
		if s >= "\x01" && s <= "\x1a" {
			return "«ctrl-" + string('A'-1+s[0]) + "»"
		}
		code := "?"
		switch s[0] {
		case '\x00':
			code = "NUL"
		case '\x1C':
			code = "FS"
		case '\x1D':
			code = "GS"
		case '\x1E':
			code = "RS"
		case '\x1F':
			code = "US"
		}

		return "«" + code + "»"
	})
}
