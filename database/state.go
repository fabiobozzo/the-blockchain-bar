package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Snapshot [32]byte

type State struct {
	Balances  map[Account]uint
	txMemPool []Tx

	dbFile          *os.File
	latestBlockHash Hash
}

func NewStateFromDisk() (*State, error) {
	state := &State{
		Balances:        map[Account]uint{},
		txMemPool:       make([]Tx, 0),
		latestBlockHash: Hash{},
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

	txFilePath := filepath.Join(cwd, "database", "block.db")
	state.dbFile, err = os.OpenFile(txFilePath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(state.dbFile)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()
		var blockFs BlockFS

		if err = json.Unmarshal(blockFsJson, &blockFs); err != nil {
			return nil, err
		}

		if err := state.applyBlock(blockFs.Value); err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
	}

	return state, nil
}

func (s *State) AddTx(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMemPool = append(s.txMemPool, tx)

	return nil
}

func (s *State) AddBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.AddTx(tx); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) Persist() (hash Hash, err error) {
	block := NewBlock(
		s.latestBlockHash,
		uint64(time.Now().Unix()),
		s.txMemPool,
	)

	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFS := BlockFS{blockHash, block}
	blockFSJson, err := json.Marshal(blockFS)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFSJson)

	if _, err := s.dbFile.Write(append(blockFSJson, '\n')); err != nil {
		return Hash{}, err
	}

	s.latestBlockHash = blockHash
	s.txMemPool = []Tx{}

	return s.latestBlockHash, nil
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}

	return nil
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
