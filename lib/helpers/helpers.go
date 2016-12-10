// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package helpers contains various routines used to help cobra
// commands stay succinct.
package helpers

import (
	"io/ioutil"
	"os"
)

// FileContentsOrStdIn returns the contents of a file, unless the file
// is "-", in which case it reads from stdin.
func FileContentsOrStdIn(s string) ([]byte, error) {
	if s == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(s)
}
