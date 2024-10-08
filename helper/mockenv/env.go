package mockenv

import (
	"blockConcur/state"
	"testing"

	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/core/types"
)

type MockEnv struct {
	Txs        []*FetchTransactions
	Headers    []*types.Header
	StateDb    []*state.IntraBlockState
	StateDbBak []*state.IntraBlockState
	DbTxs      []kv.RwTx
}

func NewMockEnv(t *testing.T, startHeight uint64) *MockEnv {
	blocks, headers, statedbs, statedb_bak, dbTxs := GetMainnetData(t, startHeight)
	return &MockEnv{
		Txs:        blocks,
		Headers:    headers,
		StateDb:    statedbs,
		StateDbBak: statedb_bak,
		DbTxs:      dbTxs,
	}
}
