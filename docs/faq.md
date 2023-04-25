# Frequently Asked Questions (FAQ)

## How do I set up a service principal for use with `ksd`?

1. First, create a service principal (if you don't have one already). Make sure you have `az` installed, then run the following commands to create a service principal:

```bash
az login # skip if already logged in
az account set --subscription <sub name or id>
az ad sp create-for-rbac --name <provide name for your application, i.e. kusto-db-syncer>
```

If `az ad sp create-for-rbac` ran successfully, you should see a valid JSON output that is similar to:

```json
{
  "appId": "6c4037f0-fdbb-4187-8816-8cb845555ed5",
  "displayName": "database-syncer",
  "password": "password",
  "tenant": "14369fg2-e0c7-4738-b54e-58f377c25f97"
}
```

Note the values of `appId`, `tenant`, and `password`.

2. Assign the service principal `users` permission to the Azure Data Explorer database. In Azure Data Explorer, with a editor tab connected to the desired cluster, run the following Kusto management commands:

```kusto
.add database <your database> users ('aadapp=<appId>;<tenant>')
```

If this succeeds, you should see a Kusto tabular result that displays the newly assigned service principal user.

3. Create a secret in your CI pipeline provider, name it something like `KUSTO_SYNC_APP_CLIENT_SECRET`, with the value of `<password>` from step 2. Modify the `ksd sync` step to use this secret value for `--client-secret`. An [example for GitHub](../examples/github/ci.yml) is available under the examples directory.

## How do I sync when multiple new functions are introduced, and the functions all reference each other?

`ksd` attempts to break dependency ties by syncing files that can be synced in a deterministic order, and retrying three times. This suffices for most situations. If needed to ensure that your functions with multiple dependencies between them will sync correctly, simply save the functions in files with names that, when sorted in ASCII order, represents the desired sync order.

For example*, if `find-and-limit.csl` depends on `find.csl`, and `find-and-limit-then-filter.csl` depends on `find-and-limit.csl`, one could name it such that the directory structure represents:s

```bash
1-find.csl
2-find-and-limit.csl
3-find-and-limit-then-filter.csl
```

In this case, `ksd` will guarantee to sync the declaration in `1-find.csl` first, followed by `2-find-and-limit.csl`, then `3-find-and-limit-then-filter.csl`.

*NOTE: This example is done purely for illustration. In reality, the example itself without modifications has filenames that when sorted in ASCII order, represent the dependency ordering.
