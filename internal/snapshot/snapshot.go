package snapshot

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"os"
)

func WriteCSV(path string, owners []DenomOwner) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"address", "amount"}); err != nil {
		log.Fatalf("failed to write CSV header: %v", err)
	}
	for _, o := range owners {
		if err := w.Write([]string{o.Address, o.Balance.Amount}); err != nil {
			log.Fatalf("failed to write CSV record: %v", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("CSV flush error: %v", err)
	}
}

func Validate(cfg Config, owners []DenomOwner) {
	snapshotTotal := new(big.Int)
	for _, o := range owners {
		amount, ok := new(big.Int).SetString(o.Balance.Amount, 10)
		if !ok {
			log.Fatalf("invalid amount %q for address %s", o.Balance.Amount, o.Address)
		}
		snapshotTotal.Add(snapshotTotal, amount)
	}

	chainSupply := fetchSupply(cfg)
	diff := new(big.Int).Sub(snapshotTotal, chainSupply)

	status := "OK"
	if diff.Sign() != 0 {
		status = "MISMATCH"
	}

	fmt.Println("Validation:")
	fmt.Printf("  Snapshot total:  %s %s\n", snapshotTotal.String(), cfg.Denom)
	fmt.Printf("  Chain supply:    %s %s\n", chainSupply.String(), cfg.Denom)
	fmt.Printf("  Difference:      %s %s (%s)\n", diff.String(), cfg.Denom, status)
}
