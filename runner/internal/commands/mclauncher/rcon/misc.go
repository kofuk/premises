package rcon

import "fmt"

func (r *Rcon) SaveAll() error {
	if _, err := r.executor.Exec("save-all"); err != nil {
		return err
	}
	return nil
}

func (r *Rcon) AddToWhiteList(player string) error {
	if _, err := r.executor.Exec(fmt.Sprintf("whitelist add %s", player)); err != nil {
		return fmt.Errorf("failed to add %s to whitelist: %w", player, err)
	}
	return nil
}

func (r *Rcon) AddToOp(player string) error {
	if _, err := r.executor.Exec(fmt.Sprintf("op %s", player)); err != nil {
		return fmt.Errorf("failed to add %s to op: %w", player, err)
	}
	return nil
}

func (r *Rcon) Say(message string) error {
	if _, err := r.executor.Exec(fmt.Sprintf("tellraw @a \"%s\"", message)); err != nil {
		return err
	}

	return nil
}

func (r *Rcon) Stop() error {
	if _, err := r.executor.Exec("stop"); err != nil {
		return err
	}
	return nil
}
