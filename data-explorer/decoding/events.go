package decoding

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"data-explorer/utils"
)

type DecodedEvent struct {
	EventName       string                 `json:"event_name"`
	ContractAddress common.Address         `json:"contract_address"`
	BlockNumber     uint64                 `json:"block_number"`
	TxHash          common.Hash            `json:"tx_hash"`
	LogIndex        uint                   `json:"log_index"`
	Topics          []common.Hash          `json:"topics"`
	Data            map[string]interface{} `json:"data"`
}

type eventMeta struct {
	name    string
	factory func() interface{}
}

var (
	eventRegistry map[common.Hash]eventMeta
	contractABI   abi.ABI
)

func init() {
	eventRegistry = make(map[common.Hash]eventMeta)
	contractABI = utils.GetABI()

	factories := map[string]func() interface{}{
		"CreateBucket":        func() interface{} { return &utils.CreateBucketEvent{} },
		"CreateFile":          func() interface{} { return &utils.CreateFileEvent{} },
		"AddFileChunk":        func() interface{} { return &utils.AddFileChunkEvent{} },
		"CommitFile":          func() interface{} { return &utils.CommitFileEvent{} },
		"FillChunkBlock":      func() interface{} { return &utils.FillChunkBlockEvent{} },
		"AddFileBlocks":       func() interface{} { return &utils.AddFileBlocksEvent{} },
		"AddPeerBlock":        func() interface{} { return &utils.AddPeerBlockEvent{} },
		"DeleteBucket":        func() interface{} { return &utils.DeleteBucketEvent{} },
		"DeletePeerBlock":     func() interface{} { return &utils.DeletePeerBlockEvent{} },
		"DeleteFile":          func() interface{} { return &utils.DeleteFileEvent{} },
		"Initialized":         func() interface{} { return &utils.InitializedEvent{} },
		"Upgraded":            func() interface{} { return &utils.UpgradedEvent{} },
		"EIP712DomainChanged": func() interface{} { return &utils.EIP712DomainChangedEvent{} },
	}

	for name, event := range contractABI.Events {
		if factory, ok := factories[name]; ok {
			eventRegistry[event.ID] = eventMeta{
				name:    name,
				factory: factory,
			}
		}
	}
}

func decodeLog(log types.Log, eventName string, out interface{}) (*DecodedEvent, error) {
	if err := utils.UnpackEvent(contractABI, out, eventName, log); err != nil {
		return nil, fmt.Errorf("failed to unpack event %s: %w", eventName, err)
	}

	decoded := &DecodedEvent{
		EventName:       eventName,
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

	topic := log.Topics[0]
	meta, ok := eventRegistry[topic]
	if !ok {
		return nil, fmt.Errorf("unknown event signature: %s", topic.Hex())
	}

	return decodeLog(log, meta.name, meta.factory())
}

func structToMap(s interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if s == nil {
		return result
	}

	val := reflect.Indirect(reflect.ValueOf(s))
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		key := field.Name
		if tag != "" && tag != "-" {
			key = tag
		}
		result[key] = val.Field(i).Interface()
	}

	return result
}
