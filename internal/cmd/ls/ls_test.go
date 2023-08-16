package ls_test

import (
	"os"

	"github.com/elishacatherasoo/rain/internal/cmd/ls"
)

func Example_ls_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	ls.Cmd.Execute()
	// Output:
	// Displays a list of all running stacks or the contents of <stack> if provided.
	//
	// Usage:
	//   ls <stack>
	//
	// Aliases:
	//   ls, list
	//
	// Flags:
	//   -a, --all    list stacks in all regions; if you specify a stack, show more details
	//   -h, --help   help for ls
}
