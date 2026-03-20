package rcon

import (
	"context"
	"fmt"
)

func (r *Rcon) SaveAll(ctx context.Context) error {
	if _, err := r.executor.Exec(ctx, "save-all"); err != nil {
		return err
	}
	return nil
}

func (r *Rcon) AddToWhiteList(ctx context.Context, player string) error {
	if _, err := r.executor.Exec(ctx, fmt.Sprintf("whitelist add %s", player)); err != nil {
		return fmt.Errorf("failed to add %s to whitelist: %w", player, err)
	}
	return nil
}

func (r *Rcon) AddToOp(ctx context.Context, player string) error {
	if _, err := r.executor.Exec(ctx, fmt.Sprintf("op %s", player)); err != nil {
		return fmt.Errorf("failed to add %s to op: %w", player, err)
	}
	return nil
}

func (r *Rcon) Say(ctx context.Context, message string) error {
	if _, err := r.executor.Exec(ctx, fmt.Sprintf("tellraw @a \"%s\"", message)); err != nil {
		return err
	}

	return nil
}

func (r *Rcon) Stop(ctx context.Context) error {
	if _, err := r.executor.Exec(ctx, "stop"); err != nil {
		return err
	}
	return nil
}
