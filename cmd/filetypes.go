// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/types"
)

var all bool // flag for whether to show all filetypes

// filetypesCmd represents the filetypes command, used to display
// valid filetypes recognized by diskii.
var filetypesCmd = &cobra.Command{
	Use:   "filetypes",
	Short: "print a list of filetypes",
	Long:  `Print a list of filetypes understood by diskii`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runFiletypes(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(filetypesCmd)
	filetypesCmd.Flags().BoolVarP(&all, "all", "a", false, "display all types, including SOS types and reserved ranges")
}

// runFiletypes performs the actual listing of filetypes.
func runFiletypes(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("filetypes expects no arguments")
	}
	for _, typ := range types.FiletypeNames(all) {
		fmt.Println(typ)
	}
	return nil
}
