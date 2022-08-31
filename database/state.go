package database

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Snapshot [32]byte

type State struct {
	Balances  map[Account]uint
	txMemPool []Tx

	dbFile   *os.File
	snapshot Snapshot
}

func NewStateFromDisk() (*State, error) {
	state := &State{
		Balances:  map[Account]uint{},
		txMemPool: make([]Tx, 0),
		snapshot:  Snapshot{},
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

	if err = state.doSnapshot(); err != nil {
		return nil, err
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

func (s *State) Persist() (snapshot Snapshot, err error) {
	memPoolCopy := make([]Tx, len(s.txMemPool))
	copy(memPoolCopy, s.txMemPool)

	for i := 0; i < len(memPoolCopy); i++ {
		txJson, err := json.Marshal(memPoolCopy[i])
		if err != nil {
			return snapshot, err
		}

		fmt.Printf("Persisting new TX to disk:\n")
		fmt.Printf("\t%s\n", txJson)
		if _, err := s.dbFile.Write(append(txJson, '\n')); err != nil {
			return snapshot, err
		}

		if err := s.doSnapshot(); err != nil {
			return snapshot, err
		}
		fmt.Printf("New DB Snapshot: %x\n", s.snapshot)

		s.txMemPool = s.txMemPool[1:]
	}

	return s.snapshot, nil
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

func (s *State) LatestSnapshot() Snapshot {
	return s.snapshot
}

func (s *State) apply(tx Tx) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value

		return nil
	}

	if s.Balances[tx.From]-tx.Value < 0 {
		return fmt.Errorf("insufficient balance for tx on account: %s", tx.From)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}

func (s *State) doSnapshot() error {
	if _, err := s.dbFile.Seek(0, 0); err != nil {
		return err
	}

	txsData, err := ioutil.ReadAll(s.dbFile)
	if err != nil {
		return err
	}

	s.snapshot = sha256.Sum256(txsData)

	return nil
}
