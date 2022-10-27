package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

const (
	TxFee             = uint(50) // TxFee is the Gas Price
	TxGas             = 21
	TxGasPriceDefault = 1
)

type State struct {
	Balances       map[common.Address]uint
	AccountToNonce map[common.Address]uint

	dbFile *os.File

	latestBlock      Block
	latestBlockHash  Hash
	hasGenesisBlock  bool
	miningDifficulty uint
	forkTIP1         uint64
}

func NewStateFromDisk(dataDir string, miningDifficulty uint) (*State, error) {
	if err := InitDataDirIfNotExists(dataDir, []byte(genesisJson)); err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	state := &State{
		Balances:         map[common.Address]uint{},
		AccountToNonce:   map[common.Address]uint{},
		latestBlockHash:  Hash{},
		latestBlock:      Block{},
		hasGenesisBlock:  false,
		miningDifficulty: miningDifficulty,
		forkTIP1:         genesis.ForkTIP1,
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
	s.miningDifficulty = pendingState.miningDifficulty

	return blockHash, nil
}

func (s *State) GetNextNonceByAccount(account common.Address) uint {
	return s.AccountToNonce[account] + 1
}

func (s *State) ChangeMiningDifficulty(newDifficulty uint) {
	s.miningDifficulty = newDifficulty
}

func (s *State) IsTIP1Fork() bool {
	return s.NextBlockNumber() >= s.forkTIP1
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
	c.miningDifficulty = s.miningDifficulty
	c.forkTIP1 = s.forkTIP1

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

	if !IsBlockHashValid(hash, s.miningDifficulty) {
		return fmt.Errorf("invalid block hash %x", hash)
	}

	if err := applyTXs(b.TXs, s); err != nil {
		return err
	}

	s.Balances[b.Header.Miner] += BlockReward

	if s.IsTIP1Fork() {
		s.Balances[b.Header.Miner] += b.GasReward()
	} else {
		s.Balances[b.Header.Miner] += uint(len(b.TXs)) * TxFee
	}

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
	if err := validateTx(tx, s); err != nil {
		return err
	}

	s.Balances[tx.From] -= tx.Cost(s.IsTIP1Fork())
	s.Balances[tx.To] += tx.Value

	s.AccountToNonce[tx.From] = tx.Nonce

	return nil
}

func validateTx(tx SignedTx, s *State) error {
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

	if s.IsTIP1Fork() {
		// Now we only have one action type, tx `transfer`, so all TXs must pay 21 gas like on Ethereum (21 000)
		if tx.Gas != TxGas {
			return fmt.Errorf("insufficient TX gas %v. required: %v", tx.Gas, TxGas)
		}

		if tx.GasPrice < TxGasPriceDefault {
			return fmt.Errorf("insufficient TX gasPrice %v. required at least: %v", tx.GasPrice, TxGasPriceDefault)
		}

	} else {
		// Prior to TIP1, a signed TX must NOT populate the Gas fields to prevent consensus from crashing
		// It's not enough to add this validation to http handlers because a TX could come from another node
		// that could modify its software and broadcast such a TX, it must be validated here too.
		if tx.Gas != 0 || tx.GasPrice != 0 {
			return fmt.Errorf("invalid TX. `Gas` and `GasPrice` can't be populate before TIP1 fork is active")
		}
	}

	if tx.Cost(s.IsTIP1Fork()) > s.Balances[tx.From] {
		return fmt.Errorf("wrong tx. sender '%s' balance is %d TBB. tx cost is %d TBB", tx.From.String(), s.Balances[tx.From], tx.Cost(s.IsTIP1Fork()))
	}

	return nil
}
