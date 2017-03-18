// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
	"github.com/zellyn/diskii/lib/helpers"
)

var filetypeName string // flag for file type
var overwrite bool      // flag for whether to overwrite

// putCmd represents the put command, used to put the raw contents
// of a file.
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "put the raw contents of a file",
	Long: `Put the raw contents of a file.

put disk-image.dsk HELLO <name of file with contents>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runPut(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(putCmd)
	putCmd.Flags().StringVarP(&filetypeName, "type", "t", "B", "Type of file (`diskii filetypes` to list)")
	putCmd.Flags().BoolVarP(&overwrite, "overwrite", "f", false, "whether to overwrite existing files")
}

// runPut performs the actual put logic.
func runPut(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: put <disk image> <target filename> <source filename>")
	}
	op, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	contents, err := helpers.FileContentsOrStdIn(args[2])
	if err != nil {
		return err
	}

	filetype, err := disk.FiletypeForName(filetypeName)
	if err != nil {
		return err
	}

	fileInfo := disk.FileInfo{
		Descriptor: disk.Descriptor{
			Name:   args[1],
			Length: len(contents),
			Type:   filetype,
		},
		Data: contents,
	}
	_, err = op.PutFile(fileInfo, overwrite)
	if err != nil {
		return err
	}
	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	_, err = op.Write(f)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
