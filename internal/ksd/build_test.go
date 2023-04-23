package ksd

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testData embed.FS

func snapshotter() *cupaloy.Config {
	return cupaloy.New(
		cupaloy.UseStringerMethods(false),
		cupaloy.EnvVariableName("UPDATE"))
}

func TestBuild_Snapshots(t *testing.T) {
	testBuild(t, "testdata/functions", "fn")
	testBuild(t, "testdata/tables", "tb")
}

func testBuild(t *testing.T, root string, prefix string) {
	ent, err := testData.ReadDir(root)
	require.NoError(t, err)
	snapshotter := snapshotter()

	for _, e := range ent {
		if e.IsDir() {
			continue
		}

		e := e
		t.Run(e.Name(), func(t *testing.T) {
			bytes, err := testData.ReadFile(filepath.Join(root, e.Name()))
			require.NoError(t, err)
			reader := strings.NewReader(string(bytes))

			decl, err := parse(reader)
			require.NoError(t, err, "parse error")

			b := &strings.Builder{}
			err = write(b, decl, filepath.Base(root))
			require.NoError(t, err, "write error")

			err = snapshotter.SnapshotWithName(
				fmt.Sprintf("%s-%s", prefix, e.Name()),
				b.String())
			require.NoError(t, err)
		})
	}
}
