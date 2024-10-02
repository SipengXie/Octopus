package pipeline

import (
	"blockConcur/eutils"
	"blockConcur/schedule"
	"blockConcur/state"
	"fmt"
	"sync"
	"time"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon/core/types"
)

type Executor struct {
	totalGas    uint64
	headers     []*types.Header
	header      *types.Header
	chainCfg    *chain.Config
	mvCache     *state.MvCache
	early_abort bool
	wg          *sync.WaitGroup
	inputChan   chan *ScheduleMessage
}

func NewExecutor(headers []*types.Header, header *types.Header,
	mvCache *state.MvCache, chainCfg *chain.Config,
	early_abort bool, wg *sync.WaitGroup,
	in chan *ScheduleMessage) *Executor {
	return &Executor{
		headers:     headers,
		header:      header,
		chainCfg:    chainCfg,
		mvCache:     mvCache,
		early_abort: early_abort,
		inputChan:   in,
		wg:          wg,
	}
}

func Execute(processors schedule.Processors, header *types.Header, headers []*types.Header, chainCfg *chain.Config, early_abort bool, mvCache *state.MvCache) (float64, uint64) {
	var wg sync.WaitGroup
	for _, processor := range processors {
		ctx := eutils.NewExecContext(header, headers, chainCfg, early_abort)
		ctx.ExecState = state.NewForRun(mvCache, header.Coinbase, early_abort)
		processor.SetExecCtx(ctx, &wg)
	}
	st := time.Now()
	for _, processor := range processors {
		wg.Add(1)
		go processor.Execute()
	}
	wg.Wait()
	mvCache.GarbageCollection()
	cost := time.Since(st).Seconds()
	totalGas := uint64(0)
	for _, processor := range processors {
		totalGas += processor.GetGas()
	}

	return cost, totalGas
}

func (e *Executor) Run() {
	var elapsed float64
	for input := range e.inputChan {
		// fmt.Println("Executor")
		if input.Flag == END {
			e.wg.Done()
			fmt.Println("Concurrent Execution Cost:", elapsed, "s")
			return
		}
		// all processors share one MVCache
		// each processor has its own cold state & exec state
		// the is a proxy to the task's read/write version
		// while the exec state maintains the localwrite
		// init execCtx for each processor
		processors := input.Processors
		cost, gas := Execute(processors, e.header, e.headers, e.chainCfg, e.early_abort, e.mvCache)
		elapsed += cost
		e.totalGas += gas
	}
}
