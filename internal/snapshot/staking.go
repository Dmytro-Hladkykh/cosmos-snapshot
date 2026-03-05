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
	"strings"
	"sync"
	"time"
)

const rewardsConcurrency = 10

type StakingEntry struct {
	DelegatorAddress string
	ValidatorAddress string
	ValidatorStatus  string
	Staked           *big.Int
	Rewards          *big.Int
}

type validatorsResponse struct {
	Validators []validatorItem `json:"validators"`
	Pagination pagination      `json:"pagination"`
}

type validatorItem struct {
	OperatorAddress string `json:"operator_address"`
	Status          string `json:"status"`
}

type delegationResponsesWrapper struct {
	DelegationResponses []delegationResp `json:"delegation_responses"`
	Pagination          pagination       `json:"pagination"`
}

type delegationResp struct {
	Delegation struct {
		DelegatorAddress string `json:"delegator_address"`
	} `json:"delegation"`
	Balance Coin `json:"balance"`
}

type decCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"` // decimal string, e.g. "1234.500000000000000000"
}

type validatorRewardsResp struct {
	Rewards []decCoin `json:"rewards"`
}

func FetchStakingSnapshot(cfg Config) []StakingEntry {
	fmt.Println("Fetching all validators...")
	validators := fetchAllValidators(cfg)
	fmt.Printf("Found %d validators\n", len(validators))

	var entries []StakingEntry
	for i, val := range validators {
		fmt.Printf("Fetching delegations for validator %d/%d (%s, %s)...\n",
			i+1, len(validators), val.OperatorAddress[:min(len(val.OperatorAddress), 20)], val.Status)
		entries = append(entries, fetchValidatorDelegations(cfg, val)...)
	}
	fmt.Printf("Total delegation entries: %d\n", len(entries))

	fmt.Printf("Fetching rewards (concurrency=%d)...\n", rewardsConcurrency)
	fetchAllRewards(cfg, entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Staked.Cmp(entries[j].Staked) > 0
	})

	return entries
}

func fetchAllValidators(cfg Config) []validatorItem {
	var all []validatorItem
	nextKey := ""

	for {
		params := url.Values{}
		params.Set("pagination.limit", "100")
		if nextKey != "" {
			params.Set("pagination.key", nextKey)
		}

		body := mustGet(cfg, "/cosmos/staking/v1beta1/validators?"+params.Encode())

		var result validatorsResponse
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("failed to parse validators response: %v", err)
		}

		all = append(all, result.Validators...)

		nextKey = result.Pagination.NextKey
		if nextKey == "" {
			break
		}
		time.Sleep(cfg.RequestDelay)
	}

	return all
}

func fetchValidatorDelegations(cfg Config, val validatorItem) []StakingEntry {
	var entries []StakingEntry
	nextKey := ""

	for {
		params := url.Values{}
		params.Set("pagination.limit", "1000")
		if nextKey != "" {
			params.Set("pagination.key", nextKey)
		}

		path := fmt.Sprintf("/cosmos/staking/v1beta1/validators/%s/delegations?%s", val.OperatorAddress, params.Encode())
		body := mustGet(cfg, path)

		var result delegationResponsesWrapper
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("failed to parse delegations response: %v", err)
		}

		for _, d := range result.DelegationResponses {
			amount, ok := new(big.Int).SetString(d.Balance.Amount, 10)
			if !ok {
				continue
			}
			entries = append(entries, StakingEntry{
				DelegatorAddress: d.Delegation.DelegatorAddress,
				ValidatorAddress: val.OperatorAddress,
				ValidatorStatus:  val.Status,
				Staked:           amount,
				Rewards:          new(big.Int),
			})
		}

		nextKey = result.Pagination.NextKey
		if nextKey == "" {
			break
		}
		time.Sleep(cfg.RequestDelay)
	}

	return entries
}

func fetchAllRewards(cfg Config, entries []StakingEntry) {
	sem := make(chan struct{}, rewardsConcurrency)
	var wg sync.WaitGroup

	done := make(chan int, len(entries))

	for i := range entries {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			time.Sleep(cfg.RequestDelay)
			entries[i].Rewards = fetchValidatorDelegatorRewards(cfg, entries[i].DelegatorAddress, entries[i].ValidatorAddress)
			done <- i
		}(i)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	finished := 0
	for range done {
		finished++
		if finished%100 == 0 {
			fmt.Printf("  Rewards: %d/%d\n", finished, len(entries))
		}
	}
}

func fetchValidatorDelegatorRewards(cfg Config, delegatorAddr, validatorAddr string) *big.Int {
	path := fmt.Sprintf("/cosmos/distribution/v1beta1/delegators/%s/rewards/%s", delegatorAddr, validatorAddr)
	body := mustGet(cfg, path)

	var result validatorRewardsResp
	if err := json.Unmarshal(body, &result); err != nil {
		return new(big.Int)
	}

	for _, c := range result.Rewards {
		if c.Denom == cfg.Denom {
			// Rewards are returned as a decimal string, e.g. "1234.500000000000000000"
			// We care abount the integer part, so floor Amount by splitting at the decimal point
			intPart := strings.SplitN(c.Amount, ".", 2)[0]
			amount, ok := new(big.Int).SetString(intPart, 10)
			if ok {
				return amount
			}
		}
	}

	return new(big.Int)
}

func WriteStakingCSV(path string, entries []StakingEntry) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("failed to create staking output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"delegator_address", "validator_address", "staked", "rewards", "status"}); err != nil {
		log.Fatalf("failed to write CSV header: %v", err)
	}
	for _, e := range entries {
		if err := w.Write([]string{e.DelegatorAddress, e.ValidatorAddress, e.Staked.String(), e.Rewards.String(), e.ValidatorStatus}); err != nil {
			log.Fatalf("failed to write CSV record: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("CSV flush error: %v", err)
	}
}
