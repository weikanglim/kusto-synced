package ksd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
