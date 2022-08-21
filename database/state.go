package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type State struct {
	Balances  map[Account]uint
	txMemPool []Tx

	dbFile *os.File
}

func NewStateFromDisk() (*State, error) {
	state := &State{
		Balances:  map[Account]uint{},
		txMemPool: make([]Tx, 0),
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	genesis, err := loadGenesisFromFile(filepath.Join(cwd, "database", "genesis.json"))
	if err != nil {
		return nil, err
	}

	for account, balance := range genesis.Balances {
		state.Balances[account] = balance
	}

	txFilePath := filepath.Join(cwd, "database", "tx.db")
	state.dbFile, err = os.OpenFile(txFilePath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(state.dbFile)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var tx Tx
		if err := json.Unmarshal(scanner.Bytes(), &tx); err != nil {
			return nil, err
		}

		if err := state.apply(tx); err != nil {
			return nil, err
		}
	}

	return state, nil
}

func (s *State) Add(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMemPool = append(s.txMemPool, tx)

	return nil
}

func (s *State) Persist() error {
	memPoolCopy := make([]Tx, len(s.txMemPool))
	copy(memPoolCopy, s.txMemPool)

	for i := 0; i < len(memPoolCopy); i++ {
		txJson, err := json.Marshal(memPoolCopy[i])
		if err != nil {
			return err
		}

		if _, err := s.dbFile.Write(append(txJson, '\n')); err != nil {
			return err
		}

		s.txMemPool = s.txMemPool[1:]
	}

	return nil
}

func (s *State) apply(tx Tx) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value

		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("insufficient balance for tx on account: %s", tx.From)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}
