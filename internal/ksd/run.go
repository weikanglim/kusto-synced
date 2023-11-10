package ksd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-kusto-go/kusto/kql"
)

// Run executes a Kusto script.
func Run(
	file string,
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
	cmdScript, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", file, err)
	}
	query := kql.New("")
	query.AddUnsafe(string(cmdScript))
	_, err = client.Mgmt(
		ctx,
		conn.db,
		query)
	if err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}
