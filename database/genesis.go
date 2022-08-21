package database

import (
	"encoding/json"
	"io/ioutil"
)

type Genesis struct {
	Balances map[Account]uint `json:"balances"`
}

func loadGenesisFromFile(path string) (Genesis, error) {
	var loadedGenesis Genesis

	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return loadedGenesis, err
	}

	err = json.Unmarshal(fileContent, &loadedGenesis)

	return loadedGenesis, err
}
