package indexing

import (
	"data-explorer/utils"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	knownFrom       int64 = 1288670
	knownTo         int64 = 1288680
	expectedMinLogs       = 900 
)

func setupClient(t *testing.T) *ethclient.Client {
	t.Helper()
	client, err := ethclient.Dial(utils.GetRPCURL())
	require.NoError(t, err, "Failed to connect to Akave RPC")
	return client
}

func TestBackfillDeterministicRange(t *testing.T) {
	client := setupClient(t)

	events, err := FetchAndDecode(client, knownFrom, knownTo)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(events), expectedMinLogs,
		"Expected at least %d events in block range [%d, %d], got %d",
		expectedMinLogs, knownFrom, knownTo, len(events))

	allowedEvents := map[string]bool{
		"CreateBucket":        true,
		"CreateFile":          true,
		"AddFileChunk":        true,
		"CommitFile":          true,
		"FillChunkBlock":      true,
		"AddFileBlocks":       true,
		"AddPeerBlock":        true,
		"DeleteBucket":        true,
		"DeletePeerBlock":     true,
		"DeleteFile":          true,
		"Initialized":         true,
		"Upgraded":            true,
		"EIP712DomainChanged": true,
	}
	for _, ev := range events {
		assert.True(t, allowedEvents[ev.EventName],
			"Unexpected event type: %s", ev.EventName)
	}

	for _, ev := range events {
		assert.GreaterOrEqual(t, ev.BlockNumber, uint64(knownFrom),
			"Event block %d is below fromBlock %d", ev.BlockNumber, knownFrom)
		assert.LessOrEqual(t, ev.BlockNumber, uint64(knownTo),
			"Event block %d is above toBlock %d", ev.BlockNumber, knownTo)
	}

	for i, ev := range events {
		assert.NotEmpty(t, ev.EventName, "Event %d has empty name", i)
		assert.NotNil(t, ev.Data, "Event %d has nil data", i)
		assert.NotEmpty(t, ev.TxHash.Hex(), "Event %d has empty TxHash", i)
	}
}

func TestBackfill_BlockOrdering(t *testing.T) {
	client := setupClient(t)

	events, err := FetchAndDecode(client, knownFrom, knownTo)
	require.NoError(t, err)
	require.NotEmpty(t, events)

	for i := 1; i < len(events); i++ {
		assert.GreaterOrEqual(t, events[i].BlockNumber, events[i-1].BlockNumber,
			"Events out of order at index %d: block %d came after block %d",
			i, events[i].BlockNumber, events[i-1].BlockNumber)
	}
}

func TestBackfill_BoundaryPrecision(t *testing.T) {
	client := setupClient(t)

	singleBlock := int64(1288680)
	events, err := FetchAndDecode(client, singleBlock, singleBlock)
	require.NoError(t, err)

	for _, ev := range events {
		assert.Equal(t, uint64(singleBlock), ev.BlockNumber,
			"Single-block query returned event from wrong block: %d", ev.BlockNumber)
	}

	adjacentBlock := singleBlock - 1
	adjacentEvents, err := FetchAndDecode(client, adjacentBlock, adjacentBlock)
	require.NoError(t, err)

	for _, ev := range adjacentEvents {
		assert.Equal(t, uint64(adjacentBlock), ev.BlockNumber,
			"Adjacent block query leaked into block: %d", ev.BlockNumber)
	}

	rangeEvents, err := FetchAndDecode(client, adjacentBlock, singleBlock)
	require.NoError(t, err)

	assert.Equal(t, len(adjacentEvents)+len(events), len(rangeEvents),
		"Boundary violation: single-block sum (%d + %d = %d) != range query (%d)",
		len(adjacentEvents), len(events),
		len(adjacentEvents)+len(events), len(rangeEvents))
}

func TestBackfill_BatchPagination(t *testing.T) {
	client := setupClient(t)

	allAtOnce, err := FetchAndDecode(client, knownFrom, knownTo)
	require.NoError(t, err)

	batched, err := FetchAndDecodeInBatches(client, knownFrom, knownTo, 2)
	require.NoError(t, err)

	assert.Equal(t, len(allAtOnce), len(batched),
		"Batch pagination lost events: single=%d, batched=%d",
		len(allAtOnce), len(batched))

	for i := range allAtOnce {
		if i >= len(batched) {
			break
		}
		assert.Equal(t, allAtOnce[i].EventName, batched[i].EventName,
			"Mismatch at index %d: single=%s, batched=%s",
			i, allAtOnce[i].EventName, batched[i].EventName)
		assert.Equal(t, allAtOnce[i].BlockNumber, batched[i].BlockNumber,
			"Block mismatch at index %d", i)
		assert.Equal(t, allAtOnce[i].TxHash, batched[i].TxHash,
			"TxHash mismatch at index %d", i)
	}
}

func TestBackfill_EventDistribution(t *testing.T) {
	client := setupClient(t)

	events, err := FetchAndDecode(client, knownFrom, knownTo)
	require.NoError(t, err)

	counts := make(map[string]int)
	for _, ev := range events {
		counts[ev.EventName]++
	}

	t.Logf("Event distribution across blocks [%d, %d]:", knownFrom, knownTo)
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("  %-20s : %d", k, counts[k])
	}

	assert.Greater(t, counts["FillChunkBlock"], 0,
		"Expected FillChunkBlock events in this range")
	assert.Greater(t, counts["AddPeerBlock"], 0,
		"Expected AddPeerBlock events in this range")

	assert.InDelta(t, counts["FillChunkBlock"], counts["AddPeerBlock"], 10,
		"FillChunkBlock and AddPeerBlock counts should be roughly equal")
}

func TestBackfill_EmptyRange(t *testing.T) {
	client := setupClient(t)

	events, err := FetchAndDecode(client, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, events, "Block 0 should have no storage contract events")
}
