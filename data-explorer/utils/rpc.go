package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"sync"
	"github.com/ethereum/go-ethereum/common"
)

// We are performing manual rpc calls instead of using the package so as to efficiently handle eth_getLogs block range errors
// basically we will map error range to messages to decode the error and if the error was rate limit exceed we retry with full range but if the error was
// block range too large we will split the range and retry with smaller ranges until we get a successful response or we hit the minimum range threshold
// please do not change the logic of this function without understanding the above point as it is crucial for the performance of the indexer and also to avoid hitting rate limits frequently
func (r *RpcUrl) MakeRequest(ctx context.Context, method string, request_id int, max_retry int, body []byte) (*JSONRPCResponse, error) {
	payload := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  json.RawMessage(body),
		ID:      request_id,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %v", err)
	}

	for i := 0; i < max_retry; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", r.GetUrl(), bytes.NewBuffer(jsonData))
			return nil, fmt.Errorf("failed to create HTTP request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP request returned non-200 status: %d", resp.StatusCode)
		}

		result := JSONRPCResponse{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("failed to decode JSON-RPC response: %v", err)
		}

		if result.Error != nil {
			if result.Error.Code >= -33000 {
				// Handle rate limit error
				fmt.Printf("Rate limit exceeded, retrying... Attempt %d/%d\n", i+1, max_retry)
				time.Sleep(time.Duration(1<<i) * time.Second) // Exponential backoff
				continue
			} else if result.Error.Code != -32005 {
				// Handle other errors
				return nil, fmt.Errorf("JSON-RPC error: %v", result.Error.Message)
			} else {
				// block range too large error, we will handle it in the calling function by splitting the range and retrying with smaller ranges
				return nil, errors.New("block range too large")
			}
		}
		return &result, nil
	}
	return nil, errors.New("max retries exceeded")
}

func (r *RpcUrl) GetLogs(ctx context.Context, request_id int, max_retry int, start int, end int, address common.Address, topics [][]string) ([]json.RawMessage, error) {
	params := []map[string]interface{}{
		{
			"fromBlock": fmt.Sprintf("0x%x", start),
			"toBlock":   fmt.Sprintf("0x%x", end),
			"address":   address.Hex(),
			"topics":    topics,
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %v", err)
	}

	resp, err := r.MakeRequest(ctx, "eth_getLogs", request_id, max_retry, body)
	if err != nil {
		if err.Error() == "block range too large" {
			// retry with smaller range by splitting the range into two halves and retrying with each half until we get a successful response or we hit the minimum range threshold
            mid := (start + end) / 2
			if mid == start || mid == end {
				return nil, fmt.Errorf("block range too large and cannot be split further: start=%d, end=%d", start, end)
			}

			var (
				wg    sync.WaitGroup
				mu    sync.Mutex
				out   []json.RawMessage
				first error
			)
				
			wg.Add(2)

			go func() {
				defer wg.Done()
				left, err := r.GetLogs(ctx, request_id, max_retry, start, mid, address, topics)
				if err != nil {
					first = err
					return
				} else {
					mu.Lock()
					out = append(out, left...)
					mu.Unlock()
				}
			}()

			go func() {
				defer wg.Done()
				right, err := r.GetLogs(ctx, request_id, max_retry, mid+1, end, address, topics)
				if err != nil {
					first = err
					return
				} else {
					mu.Lock()
					out = append(out, right...)
					mu.Unlock()
				}
			}()

			wg.Wait()

			if first != nil {
				return nil, first
			}

			return out, nil
		} else {
			return nil, err
		}
	}
	result := []json.RawMessage{}
	result = append(result, resp.Result)
	return result, nil
}
