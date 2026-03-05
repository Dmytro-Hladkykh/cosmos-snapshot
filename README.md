# cosmos-snapshot

## Description

Collects a snapshot of all native token holders on a Cosmos chain and writes it to a CSV file.

## Usage

```
export KV_VIPER_FILE=config.yaml
go run .
```

## Config

Edit `config.yaml`:

```
snapshot:
  rpc: https://rpc.example.com # Cosmos REST API URL
  denom: utoken 
  height: 0
  output: snapshot.csv
  request_delay: 1s
```

## Output

CSV file with columns: `address`, `amount` (sorted by amount descending)
