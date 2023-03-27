# Kusto Synced (ksd)

Kusto Synced (ksd) is a tool that simplifies and accelerates development for Kusto.
			
- Store commonly used Kusto functions and tables in source control. Deploy the changes using a single command locally or on CI: `ksd sync`
- Share reusable functions across teams. Functions are organized in the cluster database using the filesystem directory structure, with first-class support for adding documentation.
- Write functions and test them in Azure Data Explorer. Once you're happy, simply store it in a file. ksd automatically transpiles your User-Defined function declarations to Stored Function declarations to be saved in the database.

## Walkthrough

### Step 1: Write your function

Write your function in your favorite Kusto editor, such as [Azure Data Explorer](https://dataexplorer.azure.com/) that provides intellisense, syntax validation, and allows testing against real data.

```kusto
// Returns service requests given a time window
let ServiceRequests = (start:datetime, end:datetime) {
    Requests
    | where TIMESTAMP between(start..end)
    | extend Method = tostring(customDimensions['http.method'])
    | extend Url = tostring(customDimensions['http.url'])
    | extend StatusCode = tostring(customDimensions['http.statusCode'])
    | project TIMESTAMP, Url, Method, StatusCode, DurationMs
}
```

To test your function, simply invoke the function:

```kusto
// Returns service requests given a time window
let ServiceRequests = (start:datetime, end:datetime) {
    Requests
    | where TIMESTAMP between(start..end)
    | extend Method = tostring(customDimensions['http.method'])
    | extend Url = tostring(customDimensions['http.url'])
    | extend StatusCode = tostring(customDimensions['http.statusCode'])
    | project TIMESTAMP, Url, Method, StatusCode, DurationMs
}
ServiceRequests(ago(1d), now())
```

Once you're happy, follow the next step.

## Step 2: Save it somewhere in your machine / git repository

You may choose to organize related functions in folders that make sense for you. A general recommendation is to store functions under a `functions` folder, and table definitions under a `tables` folder. This can be under `src`, or a different folder depending on your repository.

For this example, let's assume that you have the following setup, and saved your file as `ServiceRequest.csl` in `src/functions`.

```
- <REPO>
  - src
    - functions
      - ServiceRequest.csl
```

## Step 3: Sync your Kusto functions

If you have permissions to manage the target Kusto cluster and database, simply run `ksd sync` in your `functions` or `tables` folder.

```bash
cd src/functions
ksd sync --cluster <cluster> --database <db>
```

Otherwise, add the following task to your CI pipeline:

GitHub Actions:

```yaml
- run: |
    curl <gh url>
    ksd sync src/functions
    ksd sync src/tables
```

Azure DevOps:

```yaml
- bash: |
    curl <gh url>
    ksd sync src/functions
    ksd sync src/tables
```

## Step 4: Examine new functions in the cluster

You should now be able to refresh your connection to the Azure Data Explorer, and see any new functions added:

![]()

Notice how the functions in the database is organized exactly how they are stored in source.

![]()

Also, notice that each function contains a docstring declaration that matches the comments you saved about your function.

![]()

It's that easy to write source controlled functions and tables. 

Saving Kusto functions this way helps promotes sharing and creates reusable building blocks for you and your team (think documented libraries). If your or your team practises a gated peer-review process, that would also be an added benefit. Finally, when stored in source control, you are also able to retain revision tracking, search/refactor existing declarations when changed are made to your telemetry pipeline.

## Questions?

Create or search existing issues on GitHub

### 
