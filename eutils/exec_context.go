package eutils

import (
	"blockConcur/evm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	"blockConcur/state"
	types2 "blockConcur/types"
	"fmt"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
)

// each thread execute several messages sequentially.
// multiple threads may execute different messages concurrently
type ExecContext struct {
	BlockCtx   evmtypes.BlockContext
	ChainCfg   *chain.Config
	ExecState  *state.ExecState
	EarlyAbort bool

	// for each transaction/message
	TxCtx evmtypes.TxContext
}

// header is continous, i.e., header[i+1].Number = header[i].Number + 1
func getBlockContext(headers []*types.Header, coinbase common.Address, header *types.Header) evmtypes.BlockContext {
	getHeader := func(_ common.Hash, number uint64) *types.Header {
		if number < headers[0].Number.Uint64() || number > headers[len(headers)-1].Number.Uint64() {
			panic(fmt.Sprintf("block number %d out of range", number))
			// return nil
		}
		return headers[number-headers[0].Number.Uint64()]
	}
	hashFn := evm.GetHashFn(header, getHeader)
	blkCtx := evm.NewEVMBlockContext(header, hashFn, nil, &coinbase)
	return blkCtx
}

func (ctx *ExecContext) SetTask(task *types2.Task, newRW *rwset.RwSet) {
	ctx.TxCtx = evm.NewEVMTxContext(task.Msg)
	ctx.TxCtx.TxHash = task.TxHash
	ctx.ExecState.SetTxContext(task, newRW)
}

func NewExecContext(
	// to generate block context
	header *types.Header,
	headers []*types.Header,
	chainCfg *chain.Config,
	earlyAbort bool,
) *ExecContext {

	return &ExecContext{
		BlockCtx:   getBlockContext(headers, header.Coinbase, header),
		ChainCfg:   chainCfg,
		EarlyAbort: earlyAbort,
	}
}
