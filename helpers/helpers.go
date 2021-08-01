// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// Package helpers contains helper routines for reading and writing files,
// allowing `-` to mean stdin/stdout.
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

func WriteOutput(filename string, contents []byte, force bool) error {
	if filename == "-" {
		_, err := os.Stdout.Write(contents)
		return err
	}
	if !force {
		if _, err := os.Stat(filename); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot overwrite file %q without --force (-f)", filename)
		}
	}
	return os.WriteFile(filename, contents, 0666)
}
