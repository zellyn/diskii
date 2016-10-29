// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import "github.com/spf13/cobra"

// applesoftCmd represents the applesoft command
var applesoftCmd = &cobra.Command{
	Use:   "applesoft",
	Short: "work with applesoft programs",
	Long: `diskii applesoft contains the subcommands useful for working
	with Applesoft programs.`,
}

func init() {
	RootCmd.AddCommand(applesoftCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// applesoftCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// applesoftCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
