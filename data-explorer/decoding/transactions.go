package decoding

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type DecodedTx struct {
	MethodName string                 `json:"method_name"`
	From       common.Address         `json:"from"`
	To         common.Address         `json:"to"`
	Params     map[string]interface{} `json:"params"`
	Value      *big.Int               `json:"value"`
}

func DecodeTransaction(tx *types.Transaction) (*DecodedTx, error) {
	txData := tx.Data()
	if len(txData) < 4 {
		return nil, fmt.Errorf("no method selector")
	}

	methodId := txData[:4]

	var method *abi.Method
	for _, m := range contractABI.Methods {
		if bytes.Equal(m.ID, methodId) {
			method = &m
			break
		}
	}

	if method == nil {
		return nil, fmt.Errorf("unknown method")
	}

	args := make(map[string]interface{})
	err := method.Inputs.UnpackIntoMap(args, txData[4:])
	if err != nil {
		return nil, err
	}

	var from common.Address
	signer := types.LatestSignerForChainID(tx.ChainId())
	if f, err := types.Sender(signer, tx); err == nil {
		from = f
	}

	return &DecodedTx{
		MethodName: method.Name,
		From:       from,
		To:         *tx.To(),
		Params:     args,
		Value:      tx.Value(),
	}, nil
}
