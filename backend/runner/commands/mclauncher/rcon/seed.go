package rcon

import (
	"context"
	"errors"
)

type SeedOutput string

func ParseSeedOutput(seedOutput string) (SeedOutput, error) {
	if len(seedOutput) < 8 || seedOutput[:7] != "Seed: [" || seedOutput[len(seedOutput)-1] != ']' {
		return "", errors.New("failed to retrieve seed")
	}

	return SeedOutput(seedOutput[7 : len(seedOutput)-1]), nil
}

func (r *Rcon) Seed(ctx context.Context) (SeedOutput, error) {
	seed, err := r.executor.Exec(ctx, "seed")
	if err != nil {
		return "", err
	}

	return ParseSeedOutput(seed)
}
