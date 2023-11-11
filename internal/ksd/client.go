package ksd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const (
	CredProviderGithub = "github"
)

// Credential to use to connect to a Kusto database.
// When set to the default empty struct,
// DefaultAzureCredential will be used, which typically
// relies on authentication from CLIs like `az`.
type CredentialOptions struct {
	// Authenticate using client ID.
	ClientId string
	// Tenant ID for authentication.
	TenantId string
	// Client secret.
	ClientSecret string
	// Credential provider
	CredentialProvider string
}

type kustoClient interface {
	Mgmt(ctx context.Context, db string, query kusto.Statement, options ...kusto.MgmtOption) (*kusto.RowIterator, error)
	Close() error
}

type connection struct {
	endpoint string
	db       string
}

func parseEndpoint(endpoint string) (connection, error) {
	endpointUrl, err := url.Parse(endpoint)
	if err != nil {
		return connection{}, fmt.Errorf("invalid endpoint: %w", err)
	}

	if endpointUrl.Path == "" {
		return connection{}, fmt.Errorf(
			"endpoint must target a database, and not a cluster. Does the endpoint end with the database name?")
	}

	db := strings.TrimPrefix(endpointUrl.Path, "/")
	endpointUrl.Path = ""

	return connection{
		db:       db,
		endpoint: endpointUrl.String(),
	}, nil
}

// newKustoClient creates a kusto client using the specified
// endpoint and credentials.
func newKustoClient(
	endpoint string,
	cred CredentialOptions,
	transport *http.Client) (kustoClient, error) {
	connection := kusto.NewConnectionStringBuilder(endpoint)
	if cred.ClientId != "" {
		if cred.TenantId == "" {
			return nil, errors.New("tenant id must be provided")
		}

		if cred.ClientSecret == "" && cred.CredentialProvider == "" {
			return nil, errors.New("client secret must be provided")
		}

		if cred.CredentialProvider != "" {
			cred, err := newFederatedCredential(cred.TenantId, cred.ClientId, cred.CredentialProvider)
			if err != nil {
				return nil, fmt.Errorf("creating federated credential: %w", err)
			}
			connection = connection.WithTokenCredential(cred)
		} else {
			connection = connection.WithAadAppKey(cred.ClientId, cred.ClientSecret, cred.TenantId)
		}
	} else {
		forceInteractive := false
		forceAuthEnv, has := os.LookupEnv("KSD_FORCE_INTERACTIVE_AUTH")
		if has {
			parsed, err := strconv.ParseBool(forceAuthEnv)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid value for KSD_FORCE_INTERACTIVE_AUTH: '%s'. expected truthy value: 1, true, TRUE, 0, false, FALSE", forceAuthEnv)
			}
			forceInteractive = parsed
		}
		// first, verify if azure default credential is available
		credAvailable, err := verifyDefaultAzureCredential(cred)
		if err != nil {
			log.Printf("auth: enabling interactive logon, default credential not available with error: %v", err)
		}

		if forceInteractive || !credAvailable {
			log.Println("auth: using interactive logon")
			connection = connection.WithInteractiveLogin(cred.TenantId)
		} else {
			log.Println("auth: using default credential")
			connection.AuthorityId = cred.TenantId
			connection = connection.WithDefaultAzureCredential()
		}
	}

	client, err := kusto.New(connection, kusto.WithHttpClient(transport))
	if err != nil {
		return nil, fmt.Errorf("creating kusto client: %w", err)
	}
	return client, nil
}

func newFederatedCredential(
	tenantID string,
	clientID string,
	provider string,
) (azcore.TokenCredential, error) {
	if provider != CredProviderGithub {
		return nil, fmt.Errorf("unsupported credential provider: '%s'", string(provider))
	}

	options := &azidentity.ClientAssertionCredentialOptions{}
	cred, err := azidentity.NewClientAssertionCredential(
		tenantID,
		clientID,
		func(ctx context.Context) (string, error) {
			federatedToken, err := githubToken(ctx, "api://AzureADTokenExchange")
			if err != nil {
				return "", fmt.Errorf("fetching federated token: %w", err)
			}

			return federatedToken, nil
		},
		options)
	if err != nil {
		return nil, fmt.Errorf("creating credential: %w", err)
	}

	return cred, nil
}

// githubToken gets the credential token from GitHub Actions
func githubToken(ctx context.Context, audience string) (string, error) {
	idTokenUrl, has := os.LookupEnv("ACTIONS_ID_TOKEN_REQUEST_URL")
	if !has {
		return "", errors.New("ACTIONS_ID_TOKEN_REQUEST_URL is unset")
	}

	if audience != "" {
		idTokenUrl = fmt.Sprintf("%s&audience=%s", idTokenUrl, url.QueryEscape(audience))
	}

	req, err := runtime.NewRequest(ctx, http.MethodGet, idTokenUrl)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}

	token, has := os.LookupEnv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if !has {
		return "", errors.New("ACTIONS_ID_TOKEN_REQUEST_TOKEN is unset")
	}
	req.Raw().Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := runtime.NewPipeline("", "", runtime.PipelineOptions{}, nil).Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()

	if !runtime.HasStatusCode(res, http.StatusOK) {
		return "", fmt.Errorf("expected 200 response, got: %d", res.StatusCode)
	}

	type tokenResponse struct {
		Value string `json:"value"`
	}

	tokenResp := tokenResponse{}
	err = runtime.UnmarshalAsJSON(res, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("reading body: %w", err)
	}

	if tokenResp.Value == "" {
		return "", errors.New("no token in response")
	}

	return tokenResp.Value, nil
}
