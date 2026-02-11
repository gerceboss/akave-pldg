package decoding

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"data-explorer/utils"
)

type EventSignature struct {
	Name      string
	EventSpec abi.Event
}

type DecodedEvent struct {
	EventName       string                 `json:"event_name"`
	ContractAddress common.Address         `json:"contract_address"`
	BlockNumber     uint64                 `json:"block_number"`
	TxHash          common.Hash            `json:"tx_hash"`
	LogIndex        uint                   `json:"log_index"`
	Topics          []common.Hash          `json:"topics"`
	Data            map[string]interface{} `json:"data"`
}

func decodeLog(contractABI abi.ABI, log types.Log, eventName string, out interface{}) (*DecodedEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	event, exists := contractABI.Events[eventName]
	if !exists {
		return nil, fmt.Errorf("event %s not found in ABI", eventName)
	}

	if err := utils.UnpackEvent(contractABI, out, eventName, log); err != nil {
		return nil, fmt.Errorf("failed to unpack event %s: %w", eventName, err)
	}

	decoded := &DecodedEvent{
		EventName:       event.Name,
		ContractAddress: log.Address,
		BlockNumber:     log.BlockNumber,
		TxHash:          log.TxHash,
		LogIndex:        log.Index,
		Topics:          log.Topics,
		Data:            structToMap(out),
	}

	return decoded, nil
}

func DecodeAnyLog(log types.Log) (*DecodedEvent, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	contractABI := utils.GetABI()
	topic0 := log.Topics[0]

	var eventName string
	var out interface{}

	for name, event := range contractABI.Events {
		if event.ID == topic0 {
			eventName = name
			break
		}
	}

	if eventName == "" {
		return nil, fmt.Errorf("unknown event signature: %s", topic0.Hex())
	}

	switch eventName {
	case "CreateBucket":
		out = &utils.CreateBucketEvent{}
	case "CreateFile":
		out = &utils.CreateFileEvent{}
	case "AddFileChunk":
		out = &utils.AddFileChunkEvent{}
	case "CommitFile":
		out = &utils.CommitFileEvent{}
	case "FillChunkBlock":
		out = &utils.FillChunkBlockEvent{}
	case "AddFileBlocks":
		out = &utils.AddFileBlocksEvent{}
	case "AddPeerBlock":
		out = &utils.AddPeerBlockEvent{}
	case "DeleteBucket":
		out = &utils.DeleteBucketEvent{}
	case "DeletePeerBlock":
		out = &utils.DeletePeerBlockEvent{}
	case "DeleteFile":
		out = &utils.DeleteFileEvent{}
	case "Initialized":
		out = &utils.InitializedEvent{}
	case "Upgraded":
		out = &utils.UpgradedEvent{}
	case "EIP712DomainChanged":
		out = &utils.EIP712DomainChangedEvent{}
	default:
		return nil, fmt.Errorf("decoding logic not implemented for event: %s", eventName)
	}

	return decodeLog(contractABI, log, eventName, out)
}

func structToMap(s interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	val := reflect.Indirect(reflect.ValueOf(s))
	typ := val.Type()

	if val.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldValue := val.Field(i).Interface()
		result[field.Name] = fieldValue
	}

	return result
}
