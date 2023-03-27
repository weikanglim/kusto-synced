package ksd

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*
var testData embed.FS

func snapshotter() *cupaloy.Config {
	return cupaloy.New(
		cupaloy.UseStringerMethods(false),
		cupaloy.EnvVariableName("UPDATE"))
}

func Test_build(t *testing.T) {
	test(t, "testdata/functions", "fn")
	test(t, "testdata/tables", "tb")
}

func test(t *testing.T, root string, prefix string) {
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

func Test_parse_decl(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"fn", "let x=(any){ok}"},
		{"fn_spaced", "let x = (any){ok}"},
		{"fn_spaced_newline", "let\nx\n=\n(any){ok}"},
		{"fn_comments", "//c1\n//c2\n//c3\nlet x=(any){ok}"},

		{"table", "let x=datatable(any)[ok]"},
		{"table_spaced", "let x = datatable(any) [ ok ]"},
		{"table_spaced_newline", "let\nx\n=\ndatatable(\nany\n)[\nok\n]"},
		{"table_comments", "//c1\n//c2\n//c3\nlet x=datatable(any)[ok]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decl, err := parse(strings.NewReader(tt.input))
			require.NoError(t, err)
			declType := declType(functionType)
			if strings.HasPrefix(tt.name, "table") {
				declType = tableType
			}

			assert.Equal(t, "x", decl.name)
			assert.Equal(t, declType, decl.declType)
			if declType == functionType {
				assert.Equal(t, "(any){ok}", decl.body)
			} else {
				assert.Empty(t, decl.body)
			}

			if strings.HasPrefix(tt.input, "//c1") {
				assert.Equal(t, "c1 c2 c3", decl.doc)
			} else {
				assert.Empty(t, decl.doc)
			}
		})
	}
}

func Test_parse_syntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"fn", "let x=("},
		{"fn_spaced", "let x = ( any    ) { ok }"},
		{"fn_spaced_newline", "let\nx\n=\n(\nany\n)\n{\nok\n}"},
		{"fn_filled", "let x=(anythinghereisokay"},
		{"fn_comments", "//c1\n//c2\n//c3\nlet x=("},

		{"table", "let x=datatable()[]"},
		{"table_spaced", "let x = datatable ( any ) [ ok ] "},
		{"table_spaced_newline", "let\nx\n=\ndatatable\n(\nany\n)[\n\n]"},
		{"table_filled", "let x = datatable(a,b,c,d)[any]"},
		{"table_comments", "//c1\n//c2\n//c3\nlet x=datatable()[]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse(strings.NewReader(tt.input))
			require.NoError(t, err)
		})
	}
}

func Test_parse_syntax_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"noLet", ""},
		{"noLetWithComment", "// comment"},
		{"unfinishedLet", "// comment\nlet"},
		{"unfinishedLetEqual", "// comment\nlet x"},
		{"unfinishedLetDecl", "// comment\nlet x="},

		{"invalid", "// comment\nlet x=unknown"},
		{"invalidData", "// comment\nlet x=datan"},
		{"missingTableSig", "// comment\nlet x=datatable"},
		{"missingTableClose", "// comment\nlet x=datatable("},
		{"missingTableClose", "// comment\nlet x=datatable("},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse(strings.NewReader(tt.input))
			assert.Error(t, err)
		})
	}
}
