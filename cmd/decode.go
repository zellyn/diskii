// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zellyn/diskii/lib/applesoft"
	"github.com/zellyn/diskii/lib/helpers"
)

var location uint16      // flag for starting location in memory
var rawControlCodes bool // flag for whether to skip escaping control codes

// decodeCmd represents the decode command
var decodeCmd = &cobra.Command{
	Use:   "decode filename",
	Short: "convert a binary applesoft program to a LISTing",
	Long: `
decode converts a binary Applesoft program to a text LISTing.

Examples:
decode filename # read filename
decode -        # read stdin`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDecode(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	applesoftCmd.AddCommand(decodeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// decodeCmd.PersistentFlags().String("foo", "", "A help for foo")

	decodeCmd.Flags().Uint16VarP(&location, "location", "l", 0x801, "Starting program location in memory")
	decodeCmd.Flags().BoolVarP(&rawControlCodes, "raw", "r", false, "Print raw control codes (no escaping)")
}

// runDecode performs the actual decode logic.
func runDecode(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("decode expects one argument: the filename (or - for stdin)")
	}
	contents, err := helpers.FileContentsOrStdIn(args[0])
	if err != nil {
		return err
	}
	listing, err := applesoft.Decode(contents, location)
	if err != nil {
		return err
	}
	if rawControlCodes {
		os.Stdout.WriteString(listing.String())
	} else {
		os.Stdout.WriteString(applesoft.ChevronControlCodes(listing.String()))
	}
	return nil
}
