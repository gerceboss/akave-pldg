package indexing

import (
	"context"
	"testing"

	"data-explorer/config"
	"data-explorer/decoding"

	"github.com/ethereum/go-ethereum/common"
)

func TestNoOpHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		ev   *decoding.DecodedEvent
	}{
		{"nil event", nil},
		{"empty event", &decoding.DecodedEvent{}},
		{"CreateBucket event", &decoding.DecodedEvent{
			EventName:       "CreateBucket",
			ContractAddress: common.HexToAddress("0xbFAbD47bF901ca1341D128DDD06463AA476E970B"),
			BlockNumber:     1000,
			TxHash:          common.Hash{1},
			LogIndex:        0,
			Data:            map[string]interface{}{"id": "0x01"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NoOpHandler(ctx, tt.ev); err != nil {
				t.Errorf("NoOpHandler() error = %v, want nil", err)
			}
		})
	}
}

func TestLoggingHandler(t *testing.T) {
	ctx := context.Background()

	ev := &decoding.DecodedEvent{
		EventName:       "CreateFile",
		ContractAddress: common.HexToAddress("0xbFAbD47bF901ca1341D128DDD06463AA476E970B"),
		BlockNumber:     2000,
		TxHash:          common.Hash{2},
		LogIndex:        1,
		Data:            map[string]interface{}{"bucket_id": "0xab"},
	}

	if err := LoggingHandler(ctx, ev); err != nil {
		t.Errorf("LoggingHandler() error = %v, want nil", err)
	}
}

func TestBackfill_InvalidRange(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultBackfillConfig()
	cfg.FromBlock = 1000
	cfg.ToBlock = 500 // from > to

	err := Backfill(ctx, cfg, NoOpHandler)
	if err == nil {
		t.Fatal("Backfill() expected error for invalid range, got nil")
	}
	if want := "fromBlock 1000 > toBlock 500"; err.Error() != want {
		t.Errorf("Backfill() error = %q, want %q", err.Error(), want)
	}
}

func TestBackfill_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := config.DefaultBackfillConfig()
	cfg.FromBlock = 0
	cfg.ToBlock = 0 // will try GetBlockNumber first

	err := Backfill(ctx, cfg, NoOpHandler)
	if err == nil {
		t.Fatal("Backfill() expected error with cancelled context, got nil")
	}
	// Should get context.Canceled or error from failed RPC
	if ctx.Err() == nil {
		t.Error("context should be cancelled")
	}
}

func TestBackfill_EmptyRange(t *testing.T) {
	// Use a block range that requires GetLogs; RPC will fail without network,
	// but we verify Backfill is invoked and returns an error (not panic).
	ctx := context.Background()
	cfg := config.DefaultBackfillConfig()
	cfg.RPCURL = "http://127.0.0.1:99999" // unreachable
	cfg.FromBlock = 1
	cfg.ToBlock = 1

	err := Backfill(ctx, cfg, NoOpHandler)
	if err == nil {
		t.Fatal("Backfill() expected error (no RPC), got nil")
	}
}
