// Copyright ©2021 Zellyn Hunter <zellyn@gmail.com>

package main

import (
	"reflect"
	"strconv"

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

	Ls        cmd.LsCmd        `cmd:"" aliases:"list,cat,catalog" help:"List files/directories on a disk."`
	Reorder   cmd.ReorderCmd   `cmd:"" help:"Convert between DOS-order and ProDOS-order disk images."`
	Filetypes cmd.FiletypesCmd `cmd:"" help:"Print a list of filetypes understood by diskii."`
	Put       cmd.PutCmd       `cmd:"" help:"Put the raw contents of a file onto a disk."`
	Rm        cmd.DeleteCmd    `cmd:"" aliases:"delete" help:"Delete a file."`
	Dump      cmd.DumpCmd      `cmd:"" help:"Dump the raw contents of a file."`
	Nakedos   cmd.NakedOSCmd   `cmd:"" help:"Work with NakedOS-format disks."`
	Mksd      cmd.SDCmd        `cmd:"" help:"Create a “Standard Delivery” disk image containing a binary."`
	Applesoft cmd.ApplesoftCmd `cmd:"" help:"Work with Applesoft BASIC files."`
}

func run() error {
	ctx := kong.Parse(&cli,
		kong.Name("diskii"),
		kong.Description("A commandline tool for working with Apple II disk images."),
		// kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.NamedMapper("anybaseuint16", hexUint16Mapper{}),
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

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type hexUint16Mapper struct{}

func (h hexUint16Mapper) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	t, err := ctx.Scan.PopValue("int")
	if err != nil {
		return err
	}
	var sv string
	switch v := t.Value.(type) {
	case string:
		sv = v

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		sv = fmt.Sprintf("%v", v)

	default:
		return fmt.Errorf("expected an int but got %q (%T)", t, t.Value)
	}
	n, err := strconv.ParseUint(sv, 0, 16)
	if err != nil {
		return fmt.Errorf("expected a valid %d bit uint but got %q", 16, sv)
	}
	target.SetUint(n)
	return nil
}
