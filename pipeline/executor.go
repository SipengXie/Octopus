package pipeline

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	dag "blockConcur/graph"
	occdacore "blockConcur/occda_core"
	"blockConcur/schedule"
	"blockConcur/state"
	types2 "blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
)

type Executor struct {
	totalGas    uint64
	chainCfg    *chain.Config
	mvCache     *state.MvCache
	early_abort bool
	wg          *sync.WaitGroup
	inputChan   chan *ScheduleMessage
}

func NewExecutor(mvCache *state.MvCache, chainCfg *chain.Config,
	early_abort bool, wg *sync.WaitGroup,
	in chan *ScheduleMessage) *Executor {
	return &Executor{
		chainCfg:    chainCfg,
		mvCache:     mvCache,
		early_abort: early_abort,
		inputChan:   in,
		wg:          wg,
	}
}

// process the defered tasks
// if early_abort is true, we will serial execute the defered tasks (tasks do not carry out the rwset)
// TODO: if early_abort is false, we will parallel execute the defered tasks with blockConcur, which can handle the inaccurate rwset problem
func processDeferedTasks(deferedTasks types2.Tasks, is_serial bool, use_graph bool, processor_num int, mvCache *state.MvCache, header *types.Header, headers []*types.Header, chainCfg *chain.Config) uint64 {
	totalGas := uint64(0)
	if is_serial {
		execCtx := eutils.NewExecContext(header, headers, chainCfg, false)
		execCtx.ExecState = state.NewForRun(mvCache, header.Coinbase, false)
		evm := vm.NewEVM(execCtx.BlockCtx, evmtypes.TxContext{}, execCtx.ExecState, execCtx.ChainCfg, vm.Config{})
		for _, task := range deferedTasks {
			// give task a new ID, the incarnation number will be set to 1
			task.Tid = utils.NewID(task.Tid.BlockNumber, task.Tid.TxIndex, 1)
			execCtx.SetTask(task, nil)
			evm.TxContext = execCtx.TxCtx
			msg := task.Msg
			res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
			if err == nil {
				execCtx.ExecState.Commit()
				totalGas += res.UsedGas
			}
		}
	} else {
		var graph *dag.Graph
		if use_graph {
			_, graph = GenerateGraph(deferedTasks, GenerateAccessedBy(deferedTasks))
		}
		occdaTasks := occdacore.GenerateOCCDATasks(deferedTasks)
		h_txs, tidToTaskIdx := occdacore.OCCDAInitialize(occdaTasks, graph)
		totalGas += occdacore.OCCDAMain(occdaTasks, h_txs, tidToTaskIdx, processor_num, mvCache, header, headers, chainCfg)
	}
	return totalGas
}

func Execute(processors schedule.Processors, withdraws types.Withdrawals, post_block_task *types2.Task, header *types.Header, headers []*types.Header, chainCfg *chain.Config, early_abort bool, mvCache *state.MvCache) (float64, uint64) {
	var wg sync.WaitGroup
	balanceUpdate := make(map[common.Address]*uint256.Int)
	// deal with withdrawals
	// balance update
	for _, withdrawal := range withdraws {
		balance, ok := balanceUpdate[withdrawal.Address]
		if !ok {
			balance = uint256.NewInt(0)
		}
		// amount need to be multiplied by 10^9
		factor := new(uint256.Int).SetUint64(1000000000)
		amount := new(uint256.Int).Mul(new(uint256.Int).SetUint64(withdrawal.Amount), factor)
		balance.Add(balance, amount)
		balanceUpdate[withdrawal.Address] = balance
	}

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

	var totalGas uint64
	// deal with defered tasks
	deferedTasks := make(types2.Tasks, 0)
	for _, processor := range processors {
		deferedTasks = append(deferedTasks, processor.GetDeferedTasks()...)
	}
	if len(deferedTasks) > 0 {
		// sort deferedTasks by Tid
		sort.Slice(deferedTasks, func(i, j int) bool {
			return deferedTasks[i].Tid.Less(deferedTasks[j].Tid)
		})
		totalGas += processDeferedTasks(deferedTasks, false, false, len(processors), mvCache, header, headers, chainCfg)
	}

	mvCache.GarbageCollection(balanceUpdate, post_block_task)
	cost := time.Since(st).Seconds()
	for _, processor := range processors {
		totalGas += processor.GetGas()
	}

	return cost, totalGas
}

func (e *Executor) Run() {
	var elapsed float64
	for input := range e.inputChan {
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
		cost, gas := Execute(processors, input.Withdraws, input.PostBlock, input.Header, input.Headers, e.chainCfg, e.early_abort, e.mvCache)
		elapsed += cost
		e.totalGas += gas
	}
}
