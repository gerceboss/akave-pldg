package decoding

import (
	"fmt"
	"reflect"

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

var eventRegistry map[common.Hash]eventMeta
var contractABI = utils.GetABI()

// We should add this function in main.go at the startup
func init() {
	eventRegistry = make(map[common.Hash]eventMeta)

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

	topic0 := log.Topics[0]
	meta, ok := eventRegistry[topic0]
	if !ok {
		return nil, fmt.Errorf("unknown or unsupported event signature: %s", topic0.Hex())
	}

	return decodeLog(log, meta.name, meta.factory())
}

func structToMap(s interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if s == nil {
		return result
	}

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
