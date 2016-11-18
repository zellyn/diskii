// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// ops.go contains the interfaces and helper functions for operating
// on disk images logically: catalog, rename, delete, create files,
// etc.

package disk

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Filetype describes the type of a file. It's byte-compatible with
// the ProDOS/SOS filetype byte definitions in the range 00-FF.
type Filetype int

const (
	FiletypeTypeless                Filetype = 0x00  //     | both   | Typeless file
	FiletypeBadBlocks               Filetype = 0x01  //     | both   | Bad blocks file
	FiletypeSOSPascalCode           Filetype = 0x02  //     | SOS    | PASCAL code file
	FiletypeSOSPascalText           Filetype = 0x03  //     | SOS    | PASCAL text file
	FiletypeASCIIText               Filetype = 0x04  // TXT | both   | ASCII text fil
	FiletypeSOSPascalText2          Filetype = 0x05  //     | SOS    | PASCAL text file
	FiletypeBinary                  Filetype = 0x06  // BIN | both   | Binary file
	FiletypeFont                    Filetype = 0x07  //     | SOS    | Font file
	FiletypeGraphicsScreen          Filetype = 0x08  //     | SOS    | Graphics screen file
	FiletypeBusinessBASIC           Filetype = 0x09  //     | SOS    | Business BASIC program file
	FiletypeBusinessBASICData       Filetype = 0x0A  //     | SOS    | Business BASIC data file
	FiletypeSOSWordProcessor        Filetype = 0x0B  //     | SOS    | Word processor file
	FiletypeSOSSystem               Filetype = 0x0C  //     | SOS    | SOS system file
	FiletypeDirectory               Filetype = 0x0F  // DIR | both   | Directory file
	FiletypeRPSData                 Filetype = 0x10  //     | SOS    | RPS data file
	FiletypeRPSIndex                Filetype = 0x11  //     | SOS    | RPS index file
	FiletypeAppleWorksDatabase      Filetype = 0x19  // ADB | ProDOS | AppleWorks data base file
	FiletypeAppleWorksWordProcessor Filetype = 0x1A  // AWP | ProDOS | AppleWorks word processing file
	FiletypeAppleWorksSpreadsheet   Filetype = 0x1B  // ASP | ProDOS | AppleWorks spreadsheet file
	FiletypePascal                  Filetype = 0xEF  // PAS | ProDOS | ProDOS PASCAL file
	FiletypeCommand                 Filetype = 0xF0  // CMD | ProDOS | Added command file
	FiletypeUserDefinedF1           Filetype = 0xF1  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF2           Filetype = 0xF2  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF3           Filetype = 0xF3  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF4           Filetype = 0xF4  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF5           Filetype = 0xF5  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF6           Filetype = 0xF6  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF7           Filetype = 0xF7  //     | ProDOS | ProDOS user defined file types
	FiletypeUserDefinedF8           Filetype = 0xF8  //     | ProDOS | ProDOS user defined file types
	FiletypeIntegerBASIC            Filetype = 0xFA  // INT | ProDOS | Integer BASIC program file
	FiletypeIntegerBASICVariables   Filetype = 0xFB  // IVR | ProDOS | Integer BASIC variables file
	FiletypeApplesoftBASIC          Filetype = 0xFC  // BAS | ProDOS | Applesoft BASIC program file
	FiletypeApplesoftBASICVariables Filetype = 0xFD  // VAR | ProDOS | Applesoft BASIC variables file
	FiletypeRelocatable             Filetype = 0xFE  // REL | ProDOS | EDASM relocatable object module file
	FiletypeSystem                  Filetype = 0xFF  // SYS | ProDOS | System file
	FiletypeS                       Filetype = 0x100 // DOS 3.3 Type "S"
	FiletypeA                       Filetype = 0x101 // DOS 3.3 Type "A"
	FiletypeB                       Filetype = 0x102 // DOS 3.3 Type "B"
	// | 0D-0E | SOS    | SOS reserved for future use
	// | 12-18 | SOS    | SOS reserved for future use
	// | 1C-BF | SOS    | SOS reserved for future use
	// | C0-EE | ProDOS | ProDOS reserved for future use
)

// Descriptor describes a file's characteristics.
type Descriptor struct {
	Name    string
	Sectors int
	Length  int
	Locked  bool
	Type    Filetype
}

// Operator is the interface that can operate on disks.
type Operator interface {
	// Name returns the name of the operator.
	Name() string
	// HasSubdirs returns true if the underlying operating system on the
	// disk allows subdirectories.
	HasSubdirs() bool
	// Catalog returns a catalog of disk entries. subdir should be empty
	// for operating systems that do not support subdirectories.
	Catalog(subdir string) ([]Descriptor, error)
	// GetFile retrieves a file by name.
	GetFile(filename string) (FileInfo, error)
}

// FileInfo represents a file descriptor plus the content.
type FileInfo struct {
	Descriptor   Descriptor
	Data         []byte
	StartAddress uint16
}

// operatorFactory is the type of functions that accept a SectorDisk,
// and may return an Operator interface to operate on it.
type operatorFactory func(SectorDisk) (Operator, error)

// operatorFactories is the map of currently-registered operator
// factories.
var operatorFactories map[string]operatorFactory

func init() {
	operatorFactories = make(map[string]operatorFactory)
}

// RegisterOperatorFactory registers an operator factory with the
// given name: a function that accepts a SectorDisk, and may return an
// Operator. It doesn't lock operatorFactories: it is expected to be
// called only from package `init` functions.
func RegisterOperatorFactory(name string, factory operatorFactory) {
	operatorFactories[name] = factory
}

// OperatorFor returns an Operator for the given SectorDisk, if possible.
func OperatorFor(sd SectorDisk) (Operator, error) {
	if len(operatorFactories) == 0 {
		return nil, errors.New("Cannot find an operator matching the given disk image (none registered)")
	}
	for _, factory := range operatorFactories {
		if operator, err := factory(sd); err == nil {
			return operator, nil
		}
	}
	names := make([]string, 0, len(operatorFactories))
	for name := range operatorFactories {
		names = append(names, `"`+name+`"`)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("Cannot find an operator matching the given disk image (tried %s)", strings.Join(names, ", "))
}
