// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package helpers contains various routines used to help cobra
// commands stay succinct.
package helpers

import (
	"io/ioutil"
	"os"
)

func FileContentsOrStdIn(s string) ([]byte, error) {
	if s == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(s)
}
