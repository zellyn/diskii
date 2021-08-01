// Package basic contains routines useful for both Applesoft and
// Integer BASIC.
package basic

import "regexp"

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
