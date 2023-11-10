package test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun_Live(t *testing.T) {
	cfg, err := getLiveConfig()
	if err != nil {
		t.Skip(err.Error())
	}

	runArgs := []string{"run"}
	runArgs = append(runArgs, argsFromConfig(cfg)...)

	tests := []struct {
		name   string
		args   []string
		chdir  string
		expect func(t *testing.T, r cmdResult)
	}{
		{
			"File",
			append(runArgs, filepath.Join("scripts", "show.csl")),
			"testdata",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.chdir != "" {
				Chdir(t, tt.chdir)
			}

			res := executeCmd(tt.args)
			require.NoError(t, res.Err)
		})
	}
}
