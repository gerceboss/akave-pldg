package utils

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

func UnpackEvent(contractABI abi.ABI, out interface{}, name string, vLog types.Log) error {
	// Unpack indexed and non-indexed event fields
	event := contractABI.Events[name]
	if len(vLog.Data) > 0 {
		if err := contractABI.UnpackIntoInterface(out, name, vLog.Data); err != nil {
			return err
		}
	}

	// Handle indexed parameters separately
	var indexed abi.Arguments
	for _, arg := range event.Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}

	// topic 0 is the event signature, so we skip it and start from topic 1
	if err := abi.ParseTopics(out, indexed, vLog.Topics[1:]); err != nil {
		return err
	}

	return nil
}
