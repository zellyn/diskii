// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
	_ "github.com/zellyn/diskii/lib/dos3"
	"github.com/zellyn/diskii/lib/helpers"
	_ "github.com/zellyn/diskii/lib/supermon"
)

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
}

// runPut performs the actual put logic.
func runPut(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("put expects a disk image filename, an disk-image filename, and a filename to read the contents from")
	}
	sd, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	op, err := disk.OperatorFor(sd)
	if err != nil {
		return err
	}
	contents, err := helpers.FileContentsOrStdIn(args[2])
	if err != nil {
		return err
	}

	return nil
}
