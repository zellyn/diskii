// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// filetype.go contains the Filetype type, along with routines for
// converting to and from strings.

package types

import "fmt"

// Filetype describes the type of a file. It's byte-compatible with
// the ProDOS/SOS filetype byte definitions in the range 00-FF.
type Filetype int

// Filetypes.
const (
	FiletypeTypeless                Filetype = 0x00  //     | both   | Typeless file
	FiletypeBadBlocks               Filetype = 0x01  //     | both   | Bad blocks file
	FiletypeSOSPascalCode           Filetype = 0x02  //     | SOS    | PASCAL code file
	FiletypeSOSPascalText           Filetype = 0x03  //     | SOS    | PASCAL text file
	FiletypeASCIIText               Filetype = 0x04  // TXT | both   | ASCII text file
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
	FiletypeUserDefinedF1           Filetype = 0xF1  //     | ProDOS | ProDOS user defined file type F1
	FiletypeUserDefinedF2           Filetype = 0xF2  //     | ProDOS | ProDOS user defined file type F2
	FiletypeUserDefinedF3           Filetype = 0xF3  //     | ProDOS | ProDOS user defined file type F3
	FiletypeUserDefinedF4           Filetype = 0xF4  //     | ProDOS | ProDOS user defined file type F4
	FiletypeUserDefinedF5           Filetype = 0xF5  //     | ProDOS | ProDOS user defined file type F5
	FiletypeUserDefinedF6           Filetype = 0xF6  //     | ProDOS | ProDOS user defined file type F6
	FiletypeUserDefinedF7           Filetype = 0xF7  //     | ProDOS | ProDOS user defined file type F7
	FiletypeUserDefinedF8           Filetype = 0xF8  //     | ProDOS | ProDOS user defined file type F8
	FiletypeIntegerBASIC            Filetype = 0xFA  // INT | ProDOS | Integer BASIC program file
	FiletypeIntegerBASICVariables   Filetype = 0xFB  // IVR | ProDOS | Integer BASIC variables file
	FiletypeApplesoftBASIC          Filetype = 0xFC  // BAS | ProDOS | Applesoft BASIC program file
	FiletypeApplesoftBASICVariables Filetype = 0xFD  // VAR | ProDOS | Applesoft BASIC variables file
	FiletypeRelocatable             Filetype = 0xFE  // REL | ProDOS | EDASM relocatable object module file
	FiletypeSystem                  Filetype = 0xFF  // SYS | ProDOS | System file
	FiletypeS                       Filetype = 0x100 // DOS 3.3 Type "S"
	FiletypeNewA                    Filetype = 0x101 // DOS 3.3 Type "new A"
	FiletypeNewB                    Filetype = 0x102 // DOS 3.3 Type "new B"
	// | 0D-0E | SOS    | SOS reserved for future use
	// | 12-18 | SOS    | SOS reserved for future use
	// | 1C-BF | SOS    | SOS reserved for future use
	// | C0-EE | ProDOS | ProDOS reserved for future use
)

// FiletypeInfo holds name information about filetype constants.
type FiletypeInfo struct {
	Type        Filetype // The type itself
	Name        string   // The constant name, without the "Filetype" prefix
	ThreeLetter string   // The three-letter abbreviation (ProDOS)
	OneLetter   string   // The one-letter abbreviation (DOS 3.x)
	Desc        string   // The description of the type
	Extra       bool     // If true, exclude from normal display listing

	Stringified string // (Generated) result of calling String() on the Constant
	NamesString string // (Generated) the names usable for this filetype.
}

// names of Filetype constants above
var filetypeInfos = []FiletypeInfo{
	{Type: FiletypeTypeless, Name: "Typeless", Desc: "Typeless file"},
	{Type: FiletypeBadBlocks, Name: "BadBlocks", Desc: "Bad blocks file"},
	{Type: FiletypeSOSPascalCode, Name: "SOSPascalCode", Desc: "PASCAL code file", Extra: true},
	{Type: FiletypeSOSPascalText, Name: "SOSPascalText", Desc: "PASCAL text file", Extra: true},
	{Type: FiletypeASCIIText, Name: "ASCIIText", ThreeLetter: "TXT", OneLetter: "T", Desc: "ASCII text file"},
	{Type: FiletypeSOSPascalText2, Name: "SOSPascalText2", Desc: "PASCAL text file", Extra: true},
	{Type: FiletypeBinary, Name: "Binary", ThreeLetter: "BIN", OneLetter: "B", Desc: "Binary file"},
	{Type: FiletypeFont, Name: "Font", Desc: "Font file", Extra: true},
	{Type: FiletypeGraphicsScreen, Name: "GraphicsScreen", Desc: "Graphics screen file", Extra: true},
	{Type: FiletypeBusinessBASIC, Name: "BusinessBASIC", Desc: "Business BASIC program file", Extra: true},
	{Type: FiletypeBusinessBASICData, Name: "BusinessBASICData", Desc: "Business BASIC data file", Extra: true},
	{Type: FiletypeSOSWordProcessor, Name: "SOSWordProcessor", Desc: "Word processor file", Extra: true},
	{Type: FiletypeSOSSystem, Name: "SOSSystem", Desc: "SOS system file", Extra: true},
	{Type: FiletypeDirectory, Name: "Directory", ThreeLetter: "DIR", OneLetter: "D", Desc: "Directory file"},
	{Type: FiletypeRPSData, Name: "RPSData", Desc: "RPS data file", Extra: true},
	{Type: FiletypeRPSIndex, Name: "RPSIndex", Desc: "RPS index file", Extra: true},
	{Type: FiletypeAppleWorksDatabase, Name: "AppleWorksDatabase", ThreeLetter: "ADB", Desc: "AppleWorks data base file"},
	{Type: FiletypeAppleWorksWordProcessor, Name: "AppleWorksWordProcessor", ThreeLetter: "AWP", Desc: "AppleWorks word processing file"},
	{Type: FiletypeAppleWorksSpreadsheet, Name: "AppleWorksSpreadsheet", ThreeLetter: "ASP", Desc: "AppleWorks spreadsheet file"},
	{Type: FiletypePascal, Name: "Pascal", ThreeLetter: "PAS", Desc: "ProDOS PASCAL file"},
	{Type: FiletypeCommand, Name: "Command", ThreeLetter: "CMD", Desc: "Added command file"},
	{Type: FiletypeUserDefinedF1, Name: "UserDefinedF1", Desc: "ProDOS user defined file type F1", Extra: true},
	{Type: FiletypeUserDefinedF2, Name: "UserDefinedF2", Desc: "ProDOS user defined file type F2", Extra: true},
	{Type: FiletypeUserDefinedF3, Name: "UserDefinedF3", Desc: "ProDOS user defined file type F3", Extra: true},
	{Type: FiletypeUserDefinedF4, Name: "UserDefinedF4", Desc: "ProDOS user defined file type F4", Extra: true},
	{Type: FiletypeUserDefinedF5, Name: "UserDefinedF5", Desc: "ProDOS user defined file type F5", Extra: true},
	{Type: FiletypeUserDefinedF6, Name: "UserDefinedF6", Desc: "ProDOS user defined file type F6", Extra: true},
	{Type: FiletypeUserDefinedF7, Name: "UserDefinedF7", Desc: "ProDOS user defined file type F7", Extra: true},
	{Type: FiletypeUserDefinedF8, Name: "UserDefinedF8", Desc: "ProDOS user defined file type F8", Extra: true},
	{Type: FiletypeIntegerBASIC, Name: "IntegerBASIC", ThreeLetter: "INT", OneLetter: "I", Desc: "Integer BASIC program file"},
	{Type: FiletypeIntegerBASICVariables, Name: "IntegerBASICVariables", ThreeLetter: "IVR", Desc: "Integer BASIC variables file"},
	{Type: FiletypeApplesoftBASIC, Name: "ApplesoftBASIC", ThreeLetter: "BAS", OneLetter: "A", Desc: "Applesoft BASIC program file"},
	{Type: FiletypeApplesoftBASICVariables, Name: "ApplesoftBASICVariables", ThreeLetter: "VAR", Desc: "Applesoft BASIC variables file"},
	{Type: FiletypeRelocatable, Name: "Relocatable", ThreeLetter: "REL", OneLetter: "R", Desc: "EDASM relocatable object module file"},
	{Type: FiletypeSystem, Name: "System", ThreeLetter: "SYS", Desc: "System file"},
	{Type: FiletypeS, Name: "S", OneLetter: "S", Desc: `DOS 3.3 Type "S"`},
	{Type: FiletypeNewA, Name: "NewA", OneLetter: "A", Desc: `DOS 3.3 Type "new A"`},
	{Type: FiletypeNewB, Name: "NewB", OneLetter: "B", Desc: `DOS 3.3 Type "new B"`},
}

var filetypeInfosMap map[Filetype]FiletypeInfo

func init() {
	sosReserved := []Filetype{0x0D, 0x0E, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	for i := Filetype(0x1C); i < 0xC0; i++ {
		sosReserved = append(sosReserved, i)
	}
	prodosReserved := []Filetype{}
	for i := Filetype(0xC0); i < 0xEF; i++ {
		prodosReserved = append(prodosReserved, i)
	}
	for _, typ := range sosReserved {
		info := FiletypeInfo{
			Type:        typ,
			Name:        fmt.Sprintf("SOSReserved%02X", int(typ)),
			ThreeLetter: "",
			OneLetter:   "",
			Desc:        fmt.Sprintf("SOS reserved for future use %02X", int(typ)),
			Extra:       true,
		}
		filetypeInfos = append(filetypeInfos, info)
	}
	for _, typ := range prodosReserved {
		info := FiletypeInfo{
			Type:        typ,
			Name:        fmt.Sprintf("ProDOSReserved%02X", int(typ)),
			ThreeLetter: "",
			OneLetter:   "",
			Desc:        fmt.Sprintf("ProDOS reserved for future use %02X", int(typ)),
			Extra:       true,
		}
		filetypeInfos = append(filetypeInfos, info)
	}

	seen := map[string]bool{}
	filetypeInfosMap = make(map[Filetype]FiletypeInfo, len(filetypeInfos))
	for i, info := range filetypeInfos {
		info.Stringified = info.Desc + " (" + info.Name
		info.NamesString = info.Name
		if info.ThreeLetter != "" && !seen[info.ThreeLetter] {
			info.Stringified += "|" + info.ThreeLetter
			info.NamesString += "|" + info.ThreeLetter
			seen[info.ThreeLetter] = true
		}
		if info.OneLetter != "" && info.OneLetter != info.Name && !seen[info.OneLetter] {
			info.Stringified += "|" + info.OneLetter
			info.NamesString += "|" + info.OneLetter
			seen[info.OneLetter] = true
		}
		info.Stringified += ")"

		filetypeInfos[i] = info
		filetypeInfosMap[info.Type] = info
	}
}

func (f Filetype) String() string {
	if info, found := filetypeInfosMap[f]; found {
		return info.Stringified
	}
	return fmt.Sprintf("Invalid/unknown filetype %02X", int(f))
}

// FiletypeForName returns the filetype for a full, three-letter, or
// one-letter name for a Filetype.
func FiletypeForName(name string) (Filetype, error) {
	for _, info := range filetypeInfos {
		if info.Name == name || info.ThreeLetter == name || info.OneLetter == name {
			return info.Type, nil
		}
	}
	return 0, fmt.Errorf("Unknown Filetype: %q", name)
}

// FiletypeInfos returns a list information on all filetypes.
func FiletypeInfos(all bool) []FiletypeInfo {
	if all {
		return filetypeInfos
	}
	var result []FiletypeInfo
	for _, info := range filetypeInfos {
		if !info.Extra {
			result = append(result, info)
		}
	}
	return result
}
