package indexing

import (
	"context"
	"fmt"
	"log/slog"

	"data-explorer/config"
	"data-explorer/decoding"
	"data-explorer/utils"
)

// EventHandler is called for each decoded event during backfill.
type EventHandler func(ctx context.Context, ev *decoding.DecodedEvent) error

// Backfill runs eth_getLogs with event topic filters over the block range,
// decodes each log, and invokes the handler. Uses chunked requests and
// automatic range splitting for large responses.
func Backfill(ctx context.Context, cfg config.BackfillConfig, handler EventHandler) error {
	rpc := utils.NewRpcUrl(cfg.RPCURL)
	topics := utils.EventTopicFilters()

	toBlock := cfg.ToBlock
	if toBlock == 0 {
		latest, err := rpc.GetBlockNumber(ctx, 1, cfg.MaxRetry)
		if err != nil {
			return fmt.Errorf("get latest block: %w", err)
		}
		toBlock = latest
		slog.Info("backfill using latest block", "block", latest)
	}

	from := int(cfg.FromBlock)
	to := int(toBlock)
	if from > to {
		return fmt.Errorf("fromBlock %d > toBlock %d", from, to)
	}

	chunkSize := int(cfg.ChunkSize)
	if chunkSize <= 0 {
		chunkSize = 2000
	}

	slog.Info("backfill started",
		"from", from, "to", to, "chunkSize", chunkSize,
		"contract", cfg.ContractAddress.Hex())

	for start := from; start <= to; start += chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := start + chunkSize - 1
		if end > to {
			end = to
		}

		rawLogs, err := rpc.GetLogs(ctx, 1, cfg.MaxRetry, start, end, cfg.ContractAddress, topics)
		if err != nil {
			return fmt.Errorf("GetLogs %d-%d: %w", start, end, err)
		}

		for i, raw := range rawLogs {
			vLog, err := utils.RawLogToTypesLog(raw)
			if err != nil {
				return fmt.Errorf("parse log %d in block range %d-%d: %w", i, start, end, err)
			}

			decoded, err := decoding.DecodeAnyLog(vLog)
			if err != nil {
				slog.Debug("skip non-storage log", "tx", vLog.TxHash, "topics", len(vLog.Topics), "err", err)
				continue
			}

			if err := handler(ctx, decoded); err != nil {
				return fmt.Errorf("handler for %s block %d tx %s: %w",
					decoded.EventName, decoded.BlockNumber, decoded.TxHash, err)
			}
		}

		slog.Info("backfill chunk done", "from", start, "to", end, "logs", len(rawLogs))
	}

	slog.Info("backfill completed", "from", from, "to", to)
	return nil
}

// NoOpHandler is a no-op EventHandler for testing or dry runs.
func NoOpHandler(ctx context.Context, ev *decoding.DecodedEvent) error {
	return nil
}

// LoggingHandler logs each decoded event and passes through.
func LoggingHandler(ctx context.Context, ev *decoding.DecodedEvent) error {
	slog.Info("event",
		"name", ev.EventName,
		"block", ev.BlockNumber,
		"tx", ev.TxHash,
		"logIndex", ev.LogIndex,
		"data", ev.Data)
	return nil
}
