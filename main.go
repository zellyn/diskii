// Copyright ©2021 Zellyn Hunter <zellyn@gmail.com>

package main

import (
	"github.com/zellyn/diskii/cmd"
	"github.com/zellyn/diskii/dos3"
	"github.com/zellyn/diskii/prodos"
	"github.com/zellyn/diskii/supermon"
	"github.com/zellyn/diskii/types"

	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var cli struct {
	Debug bool `kong:"short='v',help='Enable debug mode.'"`

	Ls      cmd.LsCmd      `cmd:"" aliases:"cat,catalog" help:"List paths."`
	Reorder cmd.ReorderCmd `cmd:"" help:"Reorder disk images."`
}

func run() error {

	ctx := kong.Parse(&cli,
		kong.Name("diskii"),
		kong.Description("A commandline tool for working with Apple II disk images."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	globals := &types.Globals{
		Debug: cli.Debug,
		DiskOperatorFactories: []types.OperatorFactory{
			dos3.OperatorFactory{},
			supermon.OperatorFactory{},
			prodos.OperatorFactory{},
		},
	}
	// Call the Run() method of the selected parsed command.
	return ctx.Run(globals)
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
