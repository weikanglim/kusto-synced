package examples

import (
	"io/fs"
	"ksd/internal/ksd"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	dataDir := "testdata"
	inDir := filepath.Join(dataDir, "inputs")
	expectDir := filepath.Join(dataDir, "outputs")

	temp := t.TempDir()
	err := ksd.Build(
		inDir,
		temp)
	require.NoError(t, err)
	err = filepath.WalkDir(expectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(expectDir, path)
		require.NoError(t, err)

		out := filepath.Join(temp, rel)
		exp := filepath.Join(expectDir, rel)
		if d.IsDir() {
			require.DirExists(t, out)
			return nil
		}

		require.FileExists(t, out)
		actual, err := os.ReadFile(out)
		require.NoError(t, err)

		require.FileExists(t, exp)
		expected, err := os.ReadFile(exp)
		require.NoError(t, err)

		require.Equal(t, string(expected), string(actual))
		return nil
	})

	require.NoError(t, err)
}
