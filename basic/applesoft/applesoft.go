// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package applesoft provides routines for working with Applesoft
// files.
package applesoft

import (
	"bytes"
	"fmt"
)

// TokensByCode is a map from byte value to token text.
var TokensByCode = map[byte]string{
	0x80: "END",
	0x81: "FOR",
	0x82: "NEXT",
	0x83: "DATA",
	0x84: "INPUT",
	0x85: "DEL",
	0x86: "DIM",
	0x87: "READ",
	0x88: "GR",
	0x89: "TEXT",
	0x8A: "PR #",
	0x8B: "IN #",
	0x8C: "CALL",
	0x8D: "PLOT",
	0x8E: "HLIN",
	0x8F: "VLIN",
	0x90: "HGR2",
	0x91: "HGR",
	0x92: "HCOLOR=",
	0x93: "HPLOT",
	0x94: "DRAW",
	0x95: "XDRAW",
	0x96: "HTAB",
	0x97: "HOME",
	0x98: "ROT=",
	0x99: "SCALE=",
	0x9A: "SHLOAD",
	0x9B: "TRACE",
	0x9C: "NOTRACE",
	0x9D: "NORMAL",
	0x9E: "INVERSE",
	0x9F: "FLASH",
	0xA0: "COLOR=",
	0xA1: "POP",
	0xA2: "VTAB",
	0xA3: "HIMEM:",
	0xA4: "LOMEM:",
	0xA5: "ONERR",
	0xA6: "RESUME",
	0xA7: "RECALL",
	0xA8: "STORE",
	0xA9: "SPEED=",
	0xAA: "LET",
	0xAB: "GOTO",
	0xAC: "RUN",
	0xAD: "IF",
	0xAE: "RESTORE",
	0xAF: "&",
	0xB0: "GOSUB",
	0xB1: "RETURN",
	0xB2: "REM",
	0xB3: "STOP",
	0xB4: "ON",
	0xB5: "WAIT",
	0xB6: "LOAD",
	0xB7: "SAVE",
	0xB8: "DEF FN",
	0xB9: "POKE",
	0xBA: "PRINT",
	0xBB: "CONT",
	0xBC: "LIST",
	0xBD: "CLEAR",
	0xBE: "GET",
	0xBF: "NEW",
	0xC0: "TAB",
	0xC1: "TO",
	0xC2: "FN",
	0xC3: "SPC(",
	0xC4: "THEN",
	0xC5: "AT",
	0xC6: "NOT",
	0xC7: "STEP",
	0xC8: "+",
	0xC9: "-",
	0xCA: "*",
	0xCB: "/",
	0xCC: ";",
	0xCD: "AND",
	0xCE: "OR",
	0xCF: ">",
	0xD0: "=",
	0xD1: "<",
	0xD2: "SGN",
	0xD3: "INT",
	0xD4: "ABS",
	0xD5: "USR",
	0xD6: "FRE",
	0xD7: "SCRN (",
	0xD8: "PDL",
	0xD9: "POS",
	0xDA: "SQR",
	0xDB: "RND",
	0xDC: "LOG",
	0xDD: "EXP",
	0xDE: "COS",
	0xDF: "SIN",
	0xE0: "TAN",
	0xE1: "ATN",
	0xE2: "PEEK",
	0xE3: "LEN",
	0xE4: "STR$",
	0xE5: "VAL",
	0xE6: "ASC",
	0xE7: "CHR$",
	0xE8: "LEFT$",
	0xE9: "RIGHT$",
	0xEA: "MID$",
}

// Listing holds a listing of an entire BASIC program.
type Listing []Line

// Line holds a single BASIC line, with line number and text.
type Line struct {
	Num   int
	Bytes []byte
}

// Decode turns a raw binary file into a basic program. Location
// specifies the program's location in RAM (0x801 for in-ROM Applesoft, 0x3001 for tape-loaded Applesoft).
func Decode(raw []byte, location uint16) (Listing, error) {
	// First two bytes of Applesoft files on disk are length. Let's be
	// tolerant to getting either format.
	if len(raw) >= 2 {
		size := int(raw[0]) + (256 * int(raw[1]))
		if size == len(raw)-2 || size == len(raw)-3 {
			raw = raw[2:]
		}
	}

	bounds := fmt.Sprintf("$%X to $%X", location, int(location)+len(raw))

	calcOffset := func(address int) int {
		return address - int(location)
	}
	listing := []Line{}
	last := 0 // last line number
	next := int(location)
	for next != 0 {
		ofs := calcOffset(next)
		if ofs < -1 || ofs+1 >= len(raw) {
			return nil, fmt.Errorf("line %d has next line at $%X, which is outside the input range of %s", last, next, bounds)
		}
		next = int(raw[ofs]) + 256*int(raw[ofs+1])
		ofs += 2
		if next == 0 {
			break
		}
		if ofs+1 >= len(raw) {
			if len(listing) == 0 {
				return nil, fmt.Errorf("ran out of input trying to read the first line number")
			}
			return nil, fmt.Errorf("ran out of input trying to read line number of line after %d", last)
		}
		line := Line{Num: int(raw[ofs]) + 256*int(raw[ofs+1])}
		ofs += 2
		for {
			if ofs >= len(raw) {
				return nil, fmt.Errorf("ran out of input at location $%X in line %d", ofs+int(location), line.Num)
			}
			char := raw[ofs]
			if char == 0 {
				break
			}
			if char < 0x80 {
				line.Bytes = append(line.Bytes, char)
			} else {
				token := TokensByCode[char]
				if token == "" {
					return nil, fmt.Errorf("unknown token $%X in line %d", char, line.Num)
				}
				line.Bytes = append(line.Bytes, char)
			}
			ofs++
		}
		listing = append(listing, line)
	}

	return listing, nil
}

func (l Line) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d ", l.Num)
	for _, char := range l.Bytes {
		if char < 0x80 {
			buf.WriteByte(char)
		} else {
			token := TokensByCode[char]
			buf.WriteString(" " + token + " ")
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
