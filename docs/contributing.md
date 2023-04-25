# Contributing

## Pre-requisites

- `go` (Version 1.20) - https://go.dev/

## Clone & Build

First-time:

```bash
cd <directory>
git clone https://github.com/weikanglim/ksd
cd ksd
go build
```

Recurring:

```bash
go build
```

The build will produce a `ksd` or `ksd.exe` (windows) binary from your local source copy. You can run your local copy by simply invoking `./ksd <command>` in your favorite shell.

## Tests

## Run all unit and integration tests

`go test ./...` runs all unit and integration tests by default. It will skip live tests that verify end-to-end `sync` functionality that requires a live Azure Data Explorer database connection.

## Run live tests

To run live tests, environment variables need to be set.

## With user account auth

Bash:

```bash
export KSD_TEST_DEFAULT_AUTH=1
export KSD_TEST_ENDPOINT='https://<cluster>.southcentralus.kusto.windows.net/<database>'
go test ./...`
```

PowerShell:

```powershell
$env:KSD_TEST_DEFAULT_AUTH=1
$env:KSD_TEST_ENDPOINT='https://<cluster>.southcentralus.kusto.windows.net/<database>'
go test ./...
```

## With service principal

Bash:

```bash
export KSD_TEST_CLIENT_ID='<client-id>'
export KSD_TEST_CLIENT_SECRET='<client-secret>'
export KSD_TEST_TENANT_ID='<tenant-id>'
export KSD_TEST_ENDPOINT='https://<cluster>.southcentralus.kusto.windows.net/<database>'
go test ./...`
```

PowerShell:

```powershell
$env:KSD_TEST_DEFAULT_AUTH=1
$env:KSD_TEST_CLIENT_ID='<client-id>'
$env:KSD_TEST_CLIENT_SECRET='<client-secret>'
$env:KSD_TEST_TENANT_ID='<tenant-id>'
$env:KSD_TEST_ENDPOINT='https://<cluster>.southcentralus.kusto.windows.net/<database>'
go test ./...
```

## Submitting the change

Once you're happy with the changes locally, simply submit a pull request with the changes. Code owners will review the change, approve and merge it when ready.
