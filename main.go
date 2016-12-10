// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package main

import (
	"github.com/zellyn/diskii/cmd"

	// Import disk operator factories for DOS3 and Super-Mon
	_ "github.com/zellyn/diskii/lib/dos3"
	_ "github.com/zellyn/diskii/lib/supermon"
)

func main() {
	cmd.Execute()
}
