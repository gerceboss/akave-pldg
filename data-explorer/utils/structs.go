package utils

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"time"
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

type Block struct {
	Num        int64
	Hash       []byte
	ParentHash []byte
	Timestamp  time.Time
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
	Id    common.Hash    `json:"id"`
	Name  common.Hash    `json:"name"`
	Owner common.Address `json:"owner"`
}

type CreateFileEvent struct {
	Id       common.Hash    `json:"id"`
	BucketId common.Hash    `json:"bucket_id"`
	Name     common.Hash    `json:"name"`
	Owner    common.Address `json:"owner"`
}

type AddFileChunkEvent struct {
	Id       common.Hash    `json:"id"`
	BucketId common.Hash    `json:"bucket_id"`
	Name     common.Hash    `json:"name"`
	Owner    common.Address `json:"owner"`
}

type CommitFileEvent struct {
	Id       common.Hash    `json:"id"`
	BucketId common.Hash    `json:"bucket_id"`
	Name     common.Hash    `json:"name"`
	Owner    common.Address `json:"owner"`
}

type FillChunkBlockEvent struct {
	FileId     common.Hash `json:"file_id"`
	ChunkIndex *big.Int    `json:"chunk_index"`
	BlockIndex *big.Int    `json:"block_index"`
	BlockCID   common.Hash `json:"block_cid"`
	NodeId     common.Hash `json:"node_id"`
}

type AddFileBlocksEvent struct {
	Ids    common.Hash `json:"ids"`
	FileId common.Hash `json:"file_id"`
}

type AddPeerBlockEvent struct {
	BlockId common.Hash `json:"block_id"`
	PeerId  common.Hash `json:"peer_id"`
}

type DeleteBucketEvent struct {
	Id    common.Hash    `json:"id"`
	Name  common.Hash    `json:"name"`
	Owner common.Address `json:"owner"`
}

type DeletePeerBlockEvent struct {
	BlockId common.Hash `json:"block_id"`
	PeerId  common.Hash `json:"peer_id"`
}

type DeleteFileEvent struct {
	Id       common.Hash    `json:"id"`
	BucketId common.Hash    `json:"bucket_id"`
	Name     common.Hash    `json:"name"`
	Owner    common.Address `json:"owner"`
}

type EIP712DomainChangedEvent struct {
}

type InitializedEvent struct {
	Version uint64 `json:"version"`
}

type UpgradedEvent struct {
	Implementation common.Address `json:"implementation"`
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

type EventMeta struct {
	Name    string
	Factory func() interface{}
}