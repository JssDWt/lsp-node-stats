- Modify the month in main.go for the current month (previous end becomes current start)

## BreezC
- Get the forwards from BreezC (replace the date in the filename to the current date). Save the file in the current project and replace the filename in main.go.

```bash
lightning-cli listforwards settled | gzip > breezc-listforwards-settled-2024-04-05.json.gz
```

- Get the channels from BreezC (replace the date in the filename to the current date). Save the file in the current project and replace the filename in main.go.

```bash
lightning-cli listpeerchannels | gzip > breezc-listpeerchannels-2024-04-05.json.gz
```

- Get the closed channels from BreezC (replace the date in the filename to the current date). Save the file in the current project and replace the filename in main.go.

```bash
lightning-cli listclosedchannels | gzip > breezc-listclosedchannels-2024-04-05.json.gz
```

## Run
- Run the program.
  - `go build .`
  - `./lsp-node-stats`
- Replace the query outputs from the Breez node with the actual outputs of the queries on the database.