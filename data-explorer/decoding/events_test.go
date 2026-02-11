package decoding

import (
	"math/big"
	"testing"

	"data-explorer/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestDecodeAnyLog(t *testing.T) {
	abi := utils.GetABI()

	tests := []struct {
		name      string
		eventName string
		setupLog  func() types.Log
		check     func(*testing.T, *DecodedEvent)
	}{
		{
			name:      "Decode CreateBucket",
			eventName: "CreateBucket",
			setupLog: func() types.Log {
				event := abi.Events["CreateBucket"]
				bucketId := [32]byte{1}
				nameHash := common.BytesToHash([]byte("test-bucket-hash"))
				owner := common.HexToAddress("0x123")

				topics := []common.Hash{
					event.ID,
					common.BytesToHash(bucketId[:]),
					nameHash,
					common.BytesToHash(owner.Bytes()),
				}

				return types.Log{
					Topics: topics,
					Data:   []byte{},
				}
			},
			check: func(t *testing.T, decoded *DecodedEvent) {
				if decoded.EventName != "CreateBucket" {
					t.Errorf("Expected CreateBucket, got %s", decoded.EventName)
				}
				data := decoded.Data
				idVal, idOk := data["Id"].([32]byte)
				if !idOk || idVal != [32]byte{1} {
					t.Errorf("Incorrect Id: %v", data["Id"])
				}
				nameVal, nameOk := data["Name"].(common.Hash)
				if !nameOk || nameVal != common.BytesToHash([]byte("test-bucket-hash")) {
					t.Errorf("Incorrect Name hash: %v", data["Name"])
				}
				ownerVal, ownerOk := data["Owner"].(common.Address)
				if !ownerOk || ownerVal != common.HexToAddress("0x123") {
					t.Errorf("Incorrect Owner: %v", data["Owner"])
				}
			},
		},
		{
			name:      "Decode FillChunkBlock",
			eventName: "FillChunkBlock",
			setupLog: func() types.Log {
				event := abi.Events["FillChunkBlock"]
				fileId := [32]byte{2}
				chunkIndex := big.NewInt(5)
				blockIndex := big.NewInt(3)

				blockCID := [32]byte{10}
				nodeId := [32]byte{20}

				topics := []common.Hash{
					event.ID,
					common.BytesToHash(fileId[:]),
					common.BigToHash(chunkIndex),
					common.BigToHash(blockIndex),
				}

				logData := make([]byte, 64)
				copy(logData[0:32], blockCID[:])
				copy(logData[32:64], nodeId[:])

				return types.Log{
					Topics: topics,
					Data:   logData,
				}
			},
			check: func(t *testing.T, decoded *DecodedEvent) {
				data := decoded.Data
				fileIdVal, ok := data["FileId"].([32]byte)
				if !ok || fileIdVal != [32]byte{2} {
					t.Errorf("Incorrect FileId: %v", data["FileId"])
				}
				chunkIdxVal, ok := data["ChunkIndex"].(*big.Int)
				if !ok || chunkIdxVal.Cmp(big.NewInt(5)) != 0 {
					t.Errorf("Incorrect ChunkIndex: %v", data["ChunkIndex"])
				}
				blockCIDVal, ok := data["BlockCID"].([32]byte)
				if !ok || blockCIDVal != [32]byte{10} {
					t.Errorf("Incorrect BlockCID: %v", data["BlockCID"])
				}
				nodeIdVal, ok := data["NodeId"].([32]byte)
				if !ok || nodeIdVal != [32]byte{20} {
					t.Errorf("Incorrect NodeId: %v", data["NodeId"])
				}
			},
		},
		{
			name:      "Decode AddFileBlocks",
			eventName: "AddFileBlocks",
			setupLog: func() types.Log {
				event := abi.Events["AddFileBlocks"]
				idsHash := common.HexToHash("0x999")
				fileId := [32]byte{7}

				topics := []common.Hash{
					event.ID,
					idsHash,
					common.BytesToHash(fileId[:]),
				}

				return types.Log{
					Topics: topics,
					Data:   []byte{},
				}
			},
			check: func(t *testing.T, decoded *DecodedEvent) {
				data := decoded.Data
				idsVal, ok := data["Ids"].(common.Hash)
				if !ok || idsVal != common.HexToHash("0x999") {
					t.Errorf("Incorrect Ids hash: %v", data["Ids"])
				}
				fileIdVal, ok := data["FileId"].([32]byte)
				if !ok || fileIdVal != [32]byte{7} {
					t.Errorf("Incorrect FileId: %v", data["FileId"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := tt.setupLog()
			decoded, err := DecodeAnyLog(log)
			if err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
			tt.check(t, decoded)
		})
	}
}

func TestStructToMap(t *testing.T) {
	type TestStruct struct {
		Name   string `json:"name"`
		Age    int    `json:"age"`
		Secret string `json:"-"`
		Plain  string
	}

	s := TestStruct{
		Name:   "Web3",
		Age:    30,
		Secret: "Hello",
		Plain:  "Blockchain",
	}

	result := structToMap(s)

	if val, ok := result["name"]; !ok || val != "Web3" {
		t.Errorf("JSON tag name failed")
	}
	if val, ok := result["age"]; !ok || val != 30 {
		t.Errorf("JSON tag age failed")
	}
	if _, ok := result["Secret"]; ok {
		t.Errorf("Hidden field should not be present")
	}
	if val, ok := result["Plain"]; !ok || val != "Blockchain" {
		t.Errorf("Plain field failed")
	}
}
