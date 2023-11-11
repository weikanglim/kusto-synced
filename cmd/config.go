package cmd

import (
	"errors"

	"github.com/weikanglim/ksd/internal/ksd"
)

var clientId string
var clientSecret string
var credentialProvider string
var tenantId string
var endpoint string

func GetCredentialOptionsFromFlags() (ksd.CredentialOptions, error) {
	opts := ksd.CredentialOptions{}
	if clientId != "" {
		if tenantId == "" {
			return opts, errors.New("`--tenant-id` must be set when `--client-id` is provided")
		}

		if clientSecret == "" && credentialProvider == "" {
			return opts, errors.New("`--client-secret` or `--credential-provider` must be set when `--client-id` is provided")
		}

		opts.ClientId = clientId
		opts.ClientSecret = clientSecret
		opts.TenantId = tenantId
		opts.CredentialProvider = credentialProvider
	} else {
		if clientSecret != "" {
			return opts, errors.New("`--client-id` must be set when `--client-secret` is provided")
		}

		if credentialProvider != "" {
			return opts, errors.New("`--client-id` must be set when `--credential-provider` is provided")
		}

		if tenantId != "" {
			return opts, errors.New("`--client-id` must be set when `--tenant-id` is provided")
		}
	}
	return opts, nil
}
