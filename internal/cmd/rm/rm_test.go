package rm_test

import (
	"os"

	"github.com/elishacatherasoo/rain/internal/cmd/rm"
)

func Example_rm_help() {
	os.Args = []string{
		os.Args[0],
		"--help",
	}

	rm.Cmd.Execute()
	// Output:
	// Deletes the CloudFormation stack named <stack> and waits for the action to complete.
	//
	// Usage:
	//   rm <stack>
	//
	// Aliases:
	//   rm, remove, del, delete
	//
	// Flags:
	//   -d, --detach            once removal has started, don't wait around for it to finish
	//   -h, --help              help for rm
	//       --role-arn string   ARN of an IAM role that CloudFormation should assume to remove the stack
	//   -y, --yes               don't ask questions; just delete
}
