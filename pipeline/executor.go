package pipeline

import (
	"blockConcur/eutils"
	mv "blockConcur/multiversion"
	execstate "blockConcur/state/exec_state"
	"fmt"
	"sync"
	"time"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon/core/types"
)

type Executor struct {
	headers     []*types.Header
	header      *types.Header
	chainCfg    *chain.Config
	mvCache     *mv.MvCache
	early_abort bool
	wg          *sync.WaitGroup
	inputChan   chan *ScheduleMessage
}

func NewExecutor(headers []*types.Header, header *types.Header,
	mvCache *mv.MvCache, chainCfg *chain.Config,
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

func (e *Executor) Run() {
	var elapsed int64
	for input := range e.inputChan {
		// fmt.Println("Executor")
		if input.Flag == END {
			e.wg.Done()
			fmt.Println("Concurrent Execution Cost:", elapsed, "ms")
			return
		}
		// all processors share one MVCache
		// each processor has its own cold state & exec state
		// the is a proxy to the task's read/write version
		// while the exec state maintains the localwrite
		processors := input.Processors
		// init execCtx for each processor
		var wg sync.WaitGroup
		for _, processor := range processors {
			ctx := eutils.NewExecContext(e.header, e.headers, e.chainCfg, e.early_abort)
			ctx.ExecState = execstate.NewExecStateFromMVCache(e.mvCache, e.header.Coinbase, e.early_abort)
			processor.SetExecCtx(ctx, &wg)
		}

		st := time.Now()
		for _, processor := range processors {
			wg.Add(1)
			go processor.Execute()
		}
		wg.Wait()
		elapsed += time.Since(st).Milliseconds()
	}
}
