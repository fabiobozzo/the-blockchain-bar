package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"the-blockchain-bar/utils"
)

func InitDataDirIfNotExists(dataDir string, genesis []byte) error {
	if utils.FileExist(getGenesisJsonFilePath(dataDir)) {
		return nil
	}

	dbDir := getDatabaseDirPath(dataDir)
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		return err
	}

	if err := writeGenesisToDisk(getGenesisJsonFilePath(dataDir), genesis); err != nil {
		return err
	}

	blocks := getBlocksDbFilePath(dataDir)
	if err := writeEmptyBlocksDbToDisk(blocks); err != nil {
		return err
	}

	return nil
}

func getDatabaseDirPath(dataDir string) string {
	return filepath.Join(dataDir, "database")
}

func getGenesisJsonFilePath(dataDir string) string {
	return filepath.Join(getDatabaseDirPath(dataDir), "genesis.json")
}

func getBlocksDbFilePath(dataDir string) string {
	return filepath.Join(getDatabaseDirPath(dataDir), "block.db")
}

func writeEmptyBlocksDbToDisk(path string) error {
	return ioutil.WriteFile(path, []byte(""), os.ModePerm)
}
