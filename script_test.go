package main

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func testscriptMain() int {
	main()
	return 0
}

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"diskii": testscriptMain,
	}))
}

func TestFoo(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
	})
}
