package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/weikanglim/ksd/internal/ksd"
)

func TestBuild_Errors(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{
			"DirectoryNotExist",
			[]string{"build", "doesNotExist"},
			"directory doesNotExist does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := executeCmd(tt.args)
			require.Error(t, res.Err)
			require.Contains(t, res.StdErr, tt.errMsg)
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		chdir string
	}{
		{
			"Functions",
			[]string{"build", "testdata/src/functions"},
			"",
		},
		{
			"Tables",
			[]string{"build", "testdata/src/tables"},
			"",
		},
		{
			"All",
			[]string{"build", "testdata/src"},
			"",
		},
		{
			"All_WorkingDirectory",
			[]string{"build"},
			"testdata/src",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.chdir != "" {
				Chdir(t, tt.chdir)
			}

			root, err := os.Getwd()
			require.NoError(t, err)
			if len(tt.args) > 1 {
				for _, arg := range tt.args[1:] {
					if !strings.HasPrefix(arg, "-") {
						root = arg
						break
					}
				}
			}

			res := executeCmd(tt.args)
			require.NoError(t, res.Err)

			outDir := filepath.Join(root, ksd.OutDir)
			require.DirExists(t, outDir)

			filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					if d.Name() == ksd.OutDir {
						return filepath.SkipDir
					}
					return nil
				}

				if !ksd.IsKustoSourceFile(filepath.Ext(d.Name())) {
					return nil
				}

				rel, err := filepath.Rel(root, path)
				require.NoError(t, err)

				outFile := filepath.Join(outDir, rel)
				require.FileExists(t, outFile, "file not built")

				return nil
			})
		})
	}
}
