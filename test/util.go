package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Chdir(t *testing.T, dir string) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	os.Chdir(dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(wd)
		if err != nil {
			t.Fatal(err)
		}
	})
}
