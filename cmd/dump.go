// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
)

// dumpCmd represents the dump command, used to dump the raw contents
// of a file.
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "dump the raw contents of a file",
	Long: `Dump the raw contents of a file.

dump disk-image.dsk HELLO
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDump(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(dumpCmd)
}

// runDump performs the actual dump logic.
func runDump(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("dump expects a disk image filename, and a filename")
	}
	sd, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	op, err := disk.OperatorFor(sd)
	if err != nil {
		return err
	}
	file, err := op.GetFile(args[1])
	if err != nil {
		return err
	}
	// TODO(zellyn): allow writing to files
	os.Stdout.Write(file.Data)
	return nil
}
