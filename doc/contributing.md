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

Bash:
`KSD_TEST_DEFAULT_AUTH=1 KSD_TEST_ENDPOINT='https://kvc2qknmzpybns891s6n1e.southcentralus.kusto.windows.net/MyDatabase' go test ./...`



