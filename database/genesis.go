package database

import (
	_ "embed"
	"encoding/json"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/common"
)

//go:embed genesis.json
var genesisJson string

type Genesis struct {
	Balances map[common.Address]uint `json:"balances"`
}

func loadGenesis(path string) (Genesis, error) {
	var loadedGenesis Genesis

	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return loadedGenesis, err
	}

	err = json.Unmarshal(fileContent, &loadedGenesis)

	return loadedGenesis, err
}

func writeGenesisToDisk(path string, genesis []byte) error {
	return ioutil.WriteFile(path, genesis, 0644)
}
