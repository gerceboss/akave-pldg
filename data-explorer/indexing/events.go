package indexing

import (
	"context"
	"data-explorer/decoding"
	"data-explorer/utils"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var contractAddress common.Address = utils.GetAddress()

func FetchAndDecode(client *ethclient.Client, fromBlock, toBlock int64) ([]*decoding.DecodedEvent, error) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: []common.Address{contractAddress},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %v", err)
	}

	var decodedEvents []*decoding.DecodedEvent
	for _, log := range logs {
		decoded, err := decoding.DecodeAnyLog(log)
		if err != nil {
			fmt.Printf("failed to decode log at block %d: %v\n", log.BlockNumber, err)
			continue
		}
		decodedEvents = append(decodedEvents, decoded)
	}

	return decodedEvents, nil
}

func FetchAndDecodeInBatches(client *ethclient.Client, fromBlock, toBlock, batchSize int64) ([]*decoding.DecodedEvent, error) {
	var allEvents []*decoding.DecodedEvent

	for start := fromBlock; start <= toBlock; start += batchSize + 1 {
		end := start + batchSize
		if end > toBlock {
			end = toBlock
		}

		events, err := FetchAndDecode(client, start, end)
		if err != nil {
			return nil, fmt.Errorf("batch [%d-%d] failed: %v", start, end, err)
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}
