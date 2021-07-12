// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package helpers contains various routines used to help cobra
// commands stay succinct.
package helpers

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// FileContentsOrStdIn returns the contents of a file, unless the file
// is "-", in which case it reads from stdin.
func FileContentsOrStdIn(s string) ([]byte, error) {
	if s == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(s)
}

func WriteOutput(outfilename string, contents []byte, infilename string, force bool) error {
	if outfilename == "" {
		outfilename = infilename
	}
	if outfilename == "-" {
		_, err := os.Stdout.Write(contents)
		return err
	}
	if !force {
		if _, err := os.Stat(outfilename); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot overwrite file %q without --force (-f)", outfilename)
		}
	}
	return os.WriteFile(outfilename, contents, 0666)
}
