package cmd

import (
	"errors"

	"github.com/weikanglim/ksd/internal/ksd"
)

func GetCredentialOptionsFromFlags() (ksd.CredentialOptions, error) {
	opts := ksd.CredentialOptions{}
	if clientId != "" {
		if clientSecret == "" {
			return opts, errors.New("`--client-secret` must be set when `--client-id` is provided")
		}

		if tenantId == "" {
			return opts, errors.New("`--tenant-id` must be set when `--client-id` is provided")
		}

		opts.ClientId = clientId
		opts.ClientSecret = clientSecret
		opts.TenantId = tenantId
	} else {
		if clientSecret != "" {
			return opts, errors.New("`--client-id` must be set when `--client-secret` is provided")
		}

		if tenantId != "" {
			return opts, errors.New("`--client-id` must be set when `--tenant-id` is provided")
		}
	}
	return opts, nil
}
