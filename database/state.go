package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// TxFee is the Gas Price
const TxFee = uint(50)

type State struct {
	Balances       map[common.Address]uint
	AccountToNonce map[common.Address]uint

	dbFile *os.File

	latestBlock     Block
	latestBlockHash Hash
	hasGenesisBlock bool
}

func NewStateFromDisk(dataDir string) (*State, error) {
	if err := InitDataDirIfNotExists(dataDir, []byte(genesisJson)); err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	state := &State{
		Balances:        map[common.Address]uint{},
		AccountToNonce:  map[common.Address]uint{},
		latestBlockHash: Hash{},
		latestBlock:     Block{},
		hasGenesisBlock: false,
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

		if err := applyBlock(blockFs.Value, state); err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
		state.hasGenesisBlock = true
	}

	return state, nil
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}

	return s.LatestBlock().Header.Number + 1
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

	if err := applyBlock(b, &pendingState); err != nil {
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

	fmt.Printf("\npersisting new block to disk:\n")
	fmt.Printf("\t%s\n", blockFSJson)

	if _, err := s.dbFile.Write(append(blockFSJson, '\n')); err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.AccountToNonce = pendingState.AccountToNonce
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	return blockHash, nil
}

func (s *State) GetNextNonceByAccount(account common.Address) uint {
	return s.AccountToNonce[account] + 1
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

func (s *State) copy() State {
	c := State{}
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.hasGenesisBlock = s.hasGenesisBlock
	c.Balances = make(map[common.Address]uint)
	c.AccountToNonce = make(map[common.Address]uint)

	for acc, balance := range s.Balances {
		c.Balances[acc] = balance
	}

	for acc, nonce := range s.AccountToNonce {
		c.AccountToNonce[acc] = nonce
	}

	return c
}

// applyBlock verifies if block can be added to the blockchain.
// Block metadata are verified as well as transactions within (sufficient balances, etc).
func applyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block must '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && s.latestBlockHash.Hex() != b.Header.Parent.Hex() {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}

	if !IsBlockHashValid(hash) {
		return fmt.Errorf("invalid block hash %x", hash)
	}

	if err := applyTXs(b.TXs, s); err != nil {
		return err
	}

	s.Balances[b.Header.Miner] += BlockReward
	s.Balances[b.Header.Miner] += uint(len(b.TXs)) * TxFee

	return nil
}

func applyTXs(txs []SignedTx, s *State) error {
	// sort TXs by time before applying them
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Time < txs[j].Time
	})

	for _, tx := range txs {
		err := applyTx(tx, s)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyTx(tx SignedTx, s *State) error {
	validTx, err := tx.IsAuthentic()
	if err != nil {
		return err
	}

	if !validTx {
		return fmt.Errorf("wrong tx. sender '%s' is forged", tx.From.String())
	}

	expectedNonce := s.GetNextNonceByAccount(tx.From)
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("wrong tx. sender '%s' next nonce must be '%d', not '%d'", tx.From.String(), expectedNonce, tx.Nonce)
	}

	txCost := tx.Value + TxFee
	if txCost > s.Balances[tx.From] {
		return fmt.Errorf("wrong TX. Sender '%s' balance is %d TBB. Tx cost is %d TBB", tx.From.String(), s.Balances[tx.From], tx.Value)
	}

	s.Balances[tx.From] -= txCost
	s.Balances[tx.To] += tx.Value

	s.AccountToNonce[tx.From] = expectedNonce

	return nil
}
