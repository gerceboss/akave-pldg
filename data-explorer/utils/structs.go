package utils

import (
	"encoding/json"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
)

type RpcUrl struct {
	Url string
}

func (r *RpcUrl) GetUrl() string {
	return r.Url
}

func NewRpcUrl(url string) *RpcUrl {
	return &RpcUrl{
		Url: url,
	}
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error"`
	ID      int             `json:"id"`
}

// Structs as per ABI specifications

// Event structs

type CreateBucketEvent struct {
	Id    [32]byte
	Name  common.Hash
	Owner common.Address
}

type CreateFileEvent struct {
	Id       [32]byte
	BucketId [32]byte
	Name     common.Hash
	Owner    common.Address
}

type AddFileChunkEvent struct {
	Id       [32]byte
	BucketId [32]byte
	Name     common.Hash
	Owner    common.Address
}

type CommitFileEvent struct {
	Id       [32]byte
	BucketId [32]byte
	Name     common.Hash
	Owner    common.Address
}

type FillChunkBlockEvent struct {
	FileId     [32]byte
	ChunkIndex *big.Int
	BlockIndex *big.Int
	BlockCID   [32]byte
	NodeId     [32]byte
}
