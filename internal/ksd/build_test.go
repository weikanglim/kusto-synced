package ksd

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// go:embed testdata/build/*
var buildData embed.FS

func Test_build(t *testing.T) {
	_ = buildData
	type args struct {
		reader io.Reader
		folder string
	}
	tests := []struct {
		name       string
		args       args
		wantWriter string
		wantErr    bool
	}{
		// TODO: Add test cases.
		{
			name:       "function",
			args:       args{},
			wantWriter: "",
			wantErr:    false,
		},
		{
			name:       "table",
			args:       args{},
			wantWriter: "",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			if err := build(tt.args.reader, writer, tt.args.folder); (err != nil) != tt.wantErr {
				t.Errorf("build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWriter := writer.String(); gotWriter != tt.wantWriter {
				t.Errorf("build() = %v, want %v", gotWriter, tt.wantWriter)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	// if err := Build(tt.args.srcRoot, tt.args.outRoot); (err != nil) != tt.wantErr {
	// 	t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
	// }
	dataDir := "testdata/buildAll"
	inDir := filepath.Join(dataDir, "inputs")
	expectDir := filepath.Join(dataDir, "outputs")

	temp := t.TempDir()
	err := Build(
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
