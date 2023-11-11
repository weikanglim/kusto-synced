package ksd

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Azure/azure-kusto-go/kusto/kql"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

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
	files, err := kslFiles(root)
	if err != nil {
		return err
	}

	// track files that sync successfully
	succeeded := make([]bool, len(files))
	attempt := 1

	const maxAttempts = 3

	// Skip files that fail due to a server error.
	// Retry on files that fail.
	//
	// This is a naive approach in an attempt to break ties when new function declarations
	// have dependencies between them.
	for {
		var attemptErr error
		for i, file := range files {
			if succeeded[i] {
				continue
			}

			rel, err := filepath.Rel(root, file)
			if err != nil {
				return err
			}

			cmdScript, err := os.ReadFile(file)
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
				attemptErr = fmt.Errorf("syncing file %s: %w", rel, err)
				// move to next file
				continue
			}

			succeeded[i] = true
			fmt.Printf("Synced %s\n", rel)
		}

		if attemptErr == nil {
			return nil
		}

		if attempt >= maxAttempts {
			return attemptErr
		}

		attempt++
	}
}

func kslFiles(root string) (files []string, err error) {
	files = []string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

		files = append(files, path)

		return nil
	})

	return files, err
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
