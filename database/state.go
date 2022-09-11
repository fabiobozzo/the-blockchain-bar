package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type State struct {
	Balances  map[Account]uint
	txMemPool []Tx

	dbFile *os.File

	latestBlock     Block
	latestBlockHash Hash
}

func NewStateFromDisk(dataDir string) (*State, error) {
	if err := initDataDirIfNotExists(dataDir); err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	state := &State{
		Balances:        map[Account]uint{},
		txMemPool:       make([]Tx, 0),
		latestBlockHash: Hash{},
		latestBlock:     Block{},
	}

	for account, balance := range genesis.Balances {
		state.Balances[account] = balance
	}

	dbFilepath := getBlocksDbFilePath(dataDir)
	state.dbFile, err = os.OpenFile(dbFilepath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(state.dbFile)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()
		if len(blockFsJson) == 0 {
			break
		}

		var blockFs BlockFS

		if err = json.Unmarshal(blockFsJson, &blockFs); err != nil {
			return nil, err
		}

		if err := applyTXs(blockFs.Value.TXs, state); err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
	}

	return state, nil
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) AddBlocks(blocks []Block) error {
	for _, b := range blocks {
		if _, err := s.AddBlock(b); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) AddBlock(b Block) (hash Hash, err error) {
	pendingState := s.copy()

	if err := applyBlock(b, pendingState); err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFS := BlockFS{blockHash, b}
	blockFSJson, err := json.Marshal(blockFS)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFSJson)

	if _, err := s.dbFile.Write(append(blockFSJson, '\n')); err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.latestBlockHash = blockHash
	s.latestBlock = b

	return blockHash, nil
}

func (s *State) copy() State {
	c := State{}
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.txMemPool = make([]Tx, len(s.txMemPool))
	c.Balances = make(map[Account]uint)

	for acc, balance := range s.Balances {
		c.Balances[acc] = balance
	}

	for _, tx := range s.txMemPool {
		c.txMemPool = append(c.txMemPool, tx)
	}

	return c
}

func (s *State) Close() error {
	return s.dbFile.Close()
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

// applyBlock verifies if block can be added to the blockchain.
// Block metadata are verified as well as transactions within (sufficient balances, etc).
func applyBlock(b Block, s State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	if b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block must '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	if s.latestBlock.Header.Number > 0 && reflect.DeepEqual(s.latestBlockHash, b.Header.Parent) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	return applyTXs(b.TXs, &s)
}

func applyTXs(txs []Tx, s *State) error {
	for _, tx := range txs {
		err := applyTx(tx, s)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyTx(tx Tx, s *State) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("wrong TX. Sender '%s' balance is %d TBB. Tx cost is %d TBB", tx.From, s.Balances[tx.From], tx.Value)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}
