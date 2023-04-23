package examples

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSync_Errors(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{
			"MissingDatabase",
			[]string{"sync", "--endpoint", "https://examples.kusto.windows.net"},
			"endpoint must target a database",
		},
		{
			"DirectoryNotExist",
			[]string{"sync", "dirNotExist"},
			"directory dirNotExist does not exist",
		},
		{
			"ClientAuth_MissingSecretAndTenant",
			[]string{"sync", "--client-id", "some-id"},
			"`--client-secret` must be set",
		},
		{
			"ClientAuth_MissingSecret",
			[]string{"sync", "--client-id", "some-id", "--tenant-id", "some-tenant"},
			"`--client-secret` must be set",
		},
		{
			"ClientAuth_MissingTenantId",
			[]string{"sync", "--client-id", "some-id", "--client-secret", "some-secret"},
			"`--tenant-id` must be set",
		},
		{
			"ClientAuth_MissingClientId",
			[]string{"sync", "--client-secret", "some-secret", "--tenant-id", "some-tenant"},
			"`--client-id` must be set",
		},
		{
			"ClientAuth_SecretSpecified_MissingClientId",
			[]string{"sync", "--client-secret", "some-secret"},
			"`--client-id` must be set",
		},
		{
			"ClientAuth_TenantIdSpecified_MissingClientId",
			[]string{"sync", "--tenant-id", "some-tenant"},
			"`--client-id` must be set",
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

// Live tests for sync
func TestSync_Live(t *testing.T) {
	cfg, err := getLiveConfig()
	if err != nil {
		t.Skip(err.Error())
	}

	syncArgs := []string{
		"sync",
		"--client-id",
		cfg.clientId,
		"--client-secret",
		cfg.clientSecret,
		"--tenant-id",
		cfg.tenantId,
		"--endpoint",
		cfg.endpoint,
	}

	tests := []struct {
		name   string
		args   []string
		chdir  string
		expect func(t *testing.T, r cmdResult)
	}{
		{
			"FromCurrentDirectory",
			syncArgs,
			"testdata",
			nil,
		},
		{
			"DirectorySpecified",
			append(syncArgs, "testdata"),
			"",
			nil,
		},
		{
			"FromOut",
			append(syncArgs, "--from-out", "testdata/kout"),
			"",
			func(t *testing.T, r cmdResult) {
				// Building should be skipped
				require.NotContains(t, r.StdOut, "Building files")
			},
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

type syncConfig struct {
	clientId     string
	clientSecret string
	tenantId     string
	endpoint     string
}

func getLiveConfig() (syncConfig, error) {
	clientId := os.Getenv("KSD_TEST_CLIENT_ID")
	clientSecret := os.Getenv("KSD_TEST_CLIENT_SECRET")
	tenantId := os.Getenv("KSD_TEST_TENANT_ID")
	endpoint := os.Getenv("KSD_TEST_ENDPOINT")

	if clientId == "" {
		return syncConfig{}, fmt.Errorf("skipped due to missing KSD_TEST_CLIENT_ID")
	}

	if clientSecret == "" {
		return syncConfig{}, fmt.Errorf("skipped due to missing KSD_TEST_CLIENT_SECRET")
	}

	if tenantId == "" {
		return syncConfig{}, fmt.Errorf("skipped due to missing KSD_TEST_TENANT_ID")
	}

	if endpoint == "" {
		return syncConfig{}, fmt.Errorf("skipped due to missing KSD_TEST_ENDPOINT")
	}

	return syncConfig{
		clientId:     clientId,
		clientSecret: clientSecret,
		tenantId:     tenantId,
		endpoint:     endpoint,
	}, nil
}
