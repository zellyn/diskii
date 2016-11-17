// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/disk"
	_ "github.com/zellyn/diskii/lib/dos3"
	_ "github.com/zellyn/diskii/lib/supermon"
)

// catalogCmd represents the cat command, used to catalog a disk or
// directory.
var catalogCmd = &cobra.Command{
	Use:     "catalog",
	Aliases: []string{"cat", "ls"},
	Short:   "print a list of files",
	Long:    `Catalog a disk or subdirectory.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runCat(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(catalogCmd)
}

// runCat performs the actual catalog logic.
func runCat(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("cat expects a disk image filename, and an optional subdirectory")
	}
	sd, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	op, err := disk.OperatorFor(sd)
	if err != nil {
		return err
	}
	subdir := ""
	if len(args) == 2 {
		if !op.HasSubdirs() {
			return fmt.Errorf("Disks of type %q cannot have subdirectories", op.Name())
		}
		subdir = args[1]
	}
	fds, err := op.Catalog(subdir)
	if err != nil {
		return err
	}
	for _, fd := range fds {
		fmt.Println(fd.Name)
	}
	return nil
}
