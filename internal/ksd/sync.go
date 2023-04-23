package ksd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/kql"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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
		return connection{}, fmt.Errorf("invalid --endpoint: %w", err)
	}

	if endpointUrl.Path == "" {
		return connection{}, fmt.Errorf(
			"endpoint must target a database, and not a cluster. Does the endpoint end with the database name?")
	}

	db := strings.TrimPrefix(endpointUrl.Path, "/")
	return connection{
		db:       db,
		endpoint: endpoint,
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
		if cred.ClientSecret == "" {
			return nil, errors.New("client secret must be provided")
		}

		if cred.TenantId == "" {
			return nil, errors.New("tenant id must be provided")
		}

		connection = connection.WithAadAppKey(
			cred.ClientId, cred.ClientSecret, cred.TenantId)
	} else {
		// first, verify if azure default credential is available
		credAvailable, err := verifyDefaultAzureCredential(cred)
		if err != nil {
			log.Printf("enabling interactive logon, default credential not available with error: %v", err)
		}

		if credAvailable {
			connection.AuthorityId = cred.TenantId
			connection = connection.WithDefaultAzureCredential()
		} else {
			connection = connection.WithInteractiveLogin(cred.TenantId)
		}
	}

	client, err := kusto.New(connection, kusto.WithHttpClient(transport))
	if err != nil {
		return nil, fmt.Errorf("creating kusto client: %w", err)
	}
	return client, nil
}

func Sync(
	root string,
	endpoint string,
	cred CredentialOptions,
	httpClient *http.Client) error {
	conn, err := parseEndpoint(endpoint)
	if err != nil {
		return err
	}
	client, err := newKustoClient(conn.endpoint, cred, httpClient)
	if err != nil {
		return err
	}
	defer func() error {
		err = client.Close()
		if err != nil {
			return fmt.Errorf("closing client: %w", err)
		}
		return nil
	}()

	ctx := context.Background()
	root = filepath.Clean(root)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			panic(fmt.Sprintf("calculating rel path of '%s' from root '%s: %v", path, root, err))
		}
		ext := filepath.Ext(path)
		if !IsKustoSourceFile(ext) {
			log.Printf("skipping file due to non-matching extension: %s", rel)
			return nil
		}

		cmdScript, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", rel, err)
		}
		query := kql.New("")
		query.AddUnsafe(string(cmdScript))

		_, err = client.Mgmt(
			ctx,
			conn.db,
			query)
		if err != nil {
			return fmt.Errorf("syncing file %s: %w", rel, err)
		}
		fmt.Printf("Synced %s\n", rel)

		return nil
	})
}

func verifyDefaultAzureCredential(cred CredentialOptions) (credAvailable bool, err error) {
	options := azidentity.DefaultAzureCredentialOptions{
		TenantID: cred.TenantId,
	}
	defaultCred, err := azidentity.NewDefaultAzureCredential(&options)
	if err != nil {
		return false, fmt.Errorf("constructing default credential: %w", err)
	}
	_, err = defaultCred.GetToken(
		context.Background(),
		policy.TokenRequestOptions{
			Scopes: []string{
				"https://api.kusto.windows.net/.default",
			},
		})
	if err != nil {
		return false, fmt.Errorf("getting credential: %w", err)
	}
	return true, nil
}