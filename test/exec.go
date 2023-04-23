package examples

import (
	"ksd/cmd"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type cmdResult struct {
	StdOut string
	StdErr string
	Err    error
}

func executeCmd(args []string) cmdResult {
	rootCmd := cmd.NewRootCmd()
	bstdout := &strings.Builder{}
	rootCmd.SetOut(bstdout)
	bstderr := &strings.Builder{}
	rootCmd.SetErr(bstderr)
	rootCmd.SetArgs(args)

	res := cmdResult{}
	res.Err = rootCmd.Execute()
	res.StdErr = bstderr.String()
	res.StdOut = bstdout.String()
	return res
}

// A test stub for any commands
func Test_executeCmd(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected cmdResult
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := executeCmd(tt.args)
			require.Equal(t, tt.expected, res)
		})
	}
}
