package main

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/rarimo/cosmos-snapshot/internal/snapshot"
	"gitlab.com/distributed_lab/kit/kv"
)

func main() {
	cfg := snapshot.NewConfig(kv.MustFromEnv())

	owners := snapshot.FetchDenomOwners(cfg)
	fmt.Printf("Total accounts: %d\n", len(owners))

	sort.Slice(owners, func(i, j int) bool {
		a, _ := new(big.Int).SetString(owners[i].Balance.Amount, 10)
		b, _ := new(big.Int).SetString(owners[j].Balance.Amount, 10)
		return a.Cmp(b) > 0
	})

	snapshot.WriteCSV(cfg.Output, owners)
	fmt.Printf("Snapshot written to %s\n\n", cfg.Output)

	snapshot.Validate(cfg, owners)
	fmt.Println()

	// Staking: delegations + rewards
	stakingEntries := snapshot.FetchStakingSnapshot(cfg)
	fmt.Printf("Total delegators: %d\n", len(stakingEntries))

	snapshot.WriteStakingCSV(cfg.StakingOutput, stakingEntries)
	fmt.Printf("Staking snapshot written to %s\n\n", cfg.StakingOutput)

	// Unbonding delegations
	unbondingEntries := snapshot.FetchUnbondingSnapshot(cfg)
	fmt.Printf("Total unbonding entries: %d\n", len(unbondingEntries))

	snapshot.WriteUnbondingCSV(cfg.UnbondingOutput, unbondingEntries)
	fmt.Printf("Unbonding snapshot written to %s\n", cfg.UnbondingOutput)
}
