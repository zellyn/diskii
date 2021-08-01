// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/zellyn/diskii/types"
)

// FiletypesCmd is the kong `filetypes` command.
type FiletypesCmd struct {
	All bool `kong:"help='Display all types, including SOS types and reserved ranges.'"`
}

// Run the `filetypes` command.
func (f *FiletypesCmd) Run(globals *types.Globals) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Description\tName\tThree-letter Name\tOne-letter Name")
	fmt.Fprintln(w, "-----------\t----\t-----------------\t---------------")
	for _, typ := range types.FiletypeInfos(f.All) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", typ.Desc, typ.Name, typ.ThreeLetter, typ.OneLetter)
	}
	_ = w.Flush()
	return nil
}
