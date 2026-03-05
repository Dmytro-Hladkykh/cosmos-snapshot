package snapshot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"time"
)

type DenomOwner struct {
	Address string `json:"address"`
	Balance Coin   `json:"balance"`
}

type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type denomOwnersResponse struct {
	DenomOwners []DenomOwner `json:"denom_owners"`
	Pagination  pagination   `json:"pagination"`
}

type pagination struct {
	NextKey string `json:"next_key"`
}

type supplyResponse struct {
	Supply []Coin `json:"supply"`
}

func FetchDenomOwners(cfg Config) []DenomOwner {
	fmt.Printf("Fetching denom_owners for %s...\n", cfg.Denom)

	var all []DenomOwner
	nextKey := ""
	page := 1

	for {
		params := url.Values{}
		params.Set("pagination.limit", "1000")
		if nextKey != "" {
			params.Set("pagination.key", nextKey)
		}

		body := mustGet(cfg, "/cosmos/bank/v1beta1/denom_owners/"+cfg.Denom+"?"+params.Encode())

		var result denomOwnersResponse
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("failed to parse response: %v", err)
		}

		all = append(all, result.DenomOwners...)
		nextKey = result.Pagination.NextKey

		if nextKey == "" {
			fmt.Printf("Page %d: %d accounts (done)\n", page, len(result.DenomOwners))
			break
		}
		fmt.Printf("Page %d: %d accounts (next_key: %s...)\n", page, len(result.DenomOwners), nextKey[:min(len(nextKey), 12)])
		time.Sleep(cfg.RequestDelay)
		page++
	}

	return all
}

func fetchSupply(cfg Config) *big.Int {
	body := mustGet(cfg, "/cosmos/bank/v1beta1/supply")

	var result supplyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("failed to parse supply response: %v", err)
	}

	for _, c := range result.Supply {
		if c.Denom == cfg.Denom {
			amount, ok := new(big.Int).SetString(c.Amount, 10)
			if !ok {
				log.Fatalf("invalid supply amount: %s", c.Amount)
			}
			return amount
		}
	}

	log.Fatalf("denom %q not found in supply response", cfg.Denom)
	return nil
}
