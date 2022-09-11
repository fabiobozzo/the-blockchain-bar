package database

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
)

func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	blocks := make([]Block, 0)
	shouldStartCollecting := false

	if reflect.DeepEqual(blockHash, Hash{}) {
		shouldStartCollecting = true // from first block
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		// each line represents a block
		var blockFs BlockFS
		if err := json.Unmarshal(scanner.Bytes(), &blockFs); err != nil {
			return nil, err
		}

		if shouldStartCollecting {
			blocks = append(blocks, blockFs.Value)

			continue
		}

		// start collecting blocks from the next block on
		if blockFs.Key == blockHash {
			shouldStartCollecting = true
		}
	}

	return blocks, nil
}
