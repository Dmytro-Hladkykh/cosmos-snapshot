package snapshot

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"os"
	"sort"
	"time"
)

type UnbondingEntry struct {
	DelegatorAddress string
	ValidatorAddress string
	Amount           *big.Int
	CompletionTime   string
}

type unbondingResponsesWrapper struct {
	UnbondingResponses []unbondingResponse `json:"unbonding_responses"`
	Pagination         pagination          `json:"pagination"`
}

type unbondingResponse struct {
	DelegatorAddress string           `json:"delegator_address"`
	ValidatorAddress string           `json:"validator_address"`
	Entries          []unbondingEntry `json:"entries"`
}

type unbondingEntry struct {
	Balance        string `json:"balance"`
	CompletionTime string `json:"completion_time"`
}

func FetchUnbondingSnapshot(cfg Config) []UnbondingEntry {
	fmt.Println("Fetching all validators for unbonding snapshot...")
	validators := fetchAllValidators(cfg)
	fmt.Printf("Found %d validators\n", len(validators))

	var entries []UnbondingEntry
	for i, val := range validators {
		fmt.Printf("Fetching unbonding delegations for validator %d/%d...\n", i+1, len(validators))
		entries = append(entries, fetchValidatorUnbondingDelegations(cfg, val.OperatorAddress)...)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Amount.Cmp(entries[j].Amount) > 0
	})

	return entries
}

func fetchValidatorUnbondingDelegations(cfg Config, validatorAddr string) []UnbondingEntry {
	var entries []UnbondingEntry
	nextKey := ""

	for {
		params := url.Values{}
		params.Set("pagination.limit", "1000")
		if nextKey != "" {
			params.Set("pagination.key", nextKey)
		}

		path := fmt.Sprintf("/cosmos/staking/v1beta1/validators/%s/unbonding_delegations?%s", validatorAddr, params.Encode())
		body := mustGet(cfg, path)

		var result unbondingResponsesWrapper
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("failed to parse unbonding delegations response: %v", err)
		}

		for _, r := range result.UnbondingResponses {
			for _, e := range r.Entries {
				amount, ok := new(big.Int).SetString(e.Balance, 10)
				if !ok {
					continue
				}
				entries = append(entries, UnbondingEntry{
					DelegatorAddress: r.DelegatorAddress,
					ValidatorAddress: r.ValidatorAddress,
					Amount:           amount,
					CompletionTime:   e.CompletionTime,
				})
			}
		}

		nextKey = result.Pagination.NextKey
		if nextKey == "" {
			break
		}
		time.Sleep(cfg.RequestDelay)
	}

	return entries
}

func WriteUnbondingCSV(path string, entries []UnbondingEntry) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("failed to create unbonding output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"delegator_address", "validator_address", "amount", "completion_time"}); err != nil {
		log.Fatalf("failed to write CSV header: %v", err)
	}
	for _, e := range entries {
		if err := w.Write([]string{e.DelegatorAddress, e.ValidatorAddress, e.Amount.String(), e.CompletionTime}); err != nil {
			log.Fatalf("failed to write CSV record: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("CSV flush error: %v", err)
	}
}
