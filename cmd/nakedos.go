// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import "github.com/spf13/cobra"

// nakedosCmd represents the nakedos command
var nakedosCmd = &cobra.Command{
	Use:   "nakedos",
	Short: "work with NakedOS disks",
	Long: `diskii nakedos contains the subcommands useful for working
with NakedOS (and Super-Mon) disks`,
	Aliases: []string{"supermon"},
}

func init() {
	RootCmd.AddCommand(nakedosCmd)
}
