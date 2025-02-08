package occda_core

import (
	"container/heap"
	"octopus/eutils"
	core "octopus/evm"
	"octopus/evm/vm"
	"octopus/rwset"
	"octopus/state"
	"octopus/utils"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/panjf2000/ants/v2"
)

type HeapId []*OCCDATask // heap definition, consider tid as key

func (h HeapId) Len() int           { return len(h) }
func (h HeapId) Less(i, j int) bool { return h[i].Tid.Compare(h[j].Tid) < 0 }
func (h HeapId) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *HeapId) Push(x interface{}) {
	*h = append(*h, x.(*OCCDATask))
}
func (h *HeapId) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type HeapGas []*OCCDATask // heap definition, consider gas as key

func (h HeapGas) Len() int           { return len(h) }
func (h HeapGas) Less(i, j int) bool { return h[i].Cost < h[j].Cost }
func (h HeapGas) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *HeapGas) Push(x interface{}) {
	*h = append(*h, x.(*OCCDATask))
}
func (h *HeapGas) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func OCCDAMain(occdaTasks []*OCCDATask, h_txs *HeapSid, tidToTaskIdx map[*utils.ID]int, processor_num int, mvCache *state.MvCache, header *types.Header, headers []*types.Header, chainCfg *chain.Config) (float64, uint64) {
	// =====================================================================
	next := 0 // next task in tasks to be committed
	len := len(occdaTasks)
	h_ready, h_threads, h_commit := &HeapId{}, &HeapGas{}, &HeapId{}
	gasCounter := atomic.Uint64{}
	// multi-thread pool
	var wg sync.WaitGroup
	pool, _ := ants.NewPoolWithFunc(min(processor_num, runtime.NumCPU()), func(input interface{}) {
		defer wg.Done()
		occdaTask := input.(*OCCDATask)
		occdaTask.RwSet = nil
		newRW := rwset.NewRwSet()
		execCtx := eutils.NewExecContext(header, headers, chainCfg, false) // occda won't early abort
		execCtx.ExecState = state.NewForRun(mvCache, header.Coinbase, false)
		execCtx.SetTask(&occdaTask.Task, newRW)

		msg := occdaTask.Task.Msg
		evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execCtx.ExecState, execCtx.ChainCfg, vm.Config{})
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			occdaTask.stateToCommit = execCtx.ExecState
			occdaTask.gasUsed = res.UsedGas
			gasCounter.Add(res.UsedGas)
		}
		occdaTask.RwSet = newRW
	}, ants.WithPreAlloc(true), ants.WithDisablePurge(true))
	defer pool.Release()
	// =====================================================================
	startTime := time.Now()
	for next < len {
		// moving transactions from h_txs to h_ready
		for h_txs.Len() > 0 {
			occdaTask := heap.Pop(h_txs).(*OCCDATask)
			sid_idx := tidToTaskIdx[occdaTask.sid]
			if sid_idx > next-1 {
				heap.Push(h_txs, occdaTask)
				break
			} else {
				heap.Push(h_ready, occdaTask)
			}
		}

		// moving message from h_ready to h_threads
		for h_ready.Len() > 0 && h_threads.Len() < processor_num {
			occdaTask := heap.Pop(h_ready).(*OCCDATask)
			heap.Push(h_threads, occdaTask)
		}

		// Parallel Execute Messages
		// for each task in h_threads, we construct a processor
		// and use the pool to execute it
		for h_threads.Len() > 0 {
			occdatask := heap.Pop(h_threads).(*OCCDATask)
			wg.Add(1)
			pool.Invoke(occdatask)
			heap.Push(h_commit, occdatask)
		}
		wg.Wait()

		// commit/abort the state
		for h_commit.Len() > 0 {
			occdaTask := heap.Pop(h_commit).(*OCCDATask)
			tid_idx := tidToTaskIdx[occdaTask.Tid]
			if tid_idx != next {
				heap.Push(h_commit, occdaTask)
				break
			}
			abort := false
			sid_idx := tidToTaskIdx[occdaTask.sid]
			for taskIdx := sid_idx + 1; taskIdx < tid_idx; taskIdx++ {
				prevTask := occdaTasks[taskIdx]
				if occdaTask.Depend(prevTask) {
					abort = true
					break
				}
			}
			if abort {
				occdaTask.sid = occdaTasks[tid_idx-1].Tid
				heap.Push(h_txs, occdaTask)
			} else {
				// commit corresponding state
				stateToCommit := occdaTask.stateToCommit
				if stateToCommit != nil {
					stateToCommit.Commit()
				}
				next++
			}
		}
	}
	return time.Since(startTime).Seconds(), gasCounter.Load()
}
