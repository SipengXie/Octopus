package helper

import (
	"octopus/types"
	"octopus/utils"
	"runtime"
	"sync"

	types2 "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/panjf2000/ants/v2"
)

func ConvertTxToTasks(txs types2.Transactions, header *types2.Header, thread_num int) []*types.Task {

	// parallel generate messages
	cfg := params.MainnetChainConfig
	rule := cfg.Rules(header.Number.Uint64(), header.Time)
	tasks := make([]*types.Task, len(txs))
	var wg sync.WaitGroup
	bHash := header.Hash()
	number := header.Number.Uint64()

	pool, _ := ants.NewPoolWithFunc(min(thread_num, runtime.NumCPU()), func(i interface{}) {
		defer wg.Done()
		input := i.(struct {
			index int
			tx    types2.Transaction
		})
		msg, err := input.tx.AsMessage(*types2.LatestSigner(cfg), header.BaseFee, rule)
		if err != nil {
			panic(err)
		}
		globalId := utils.NewID(number, input.index, 0)
		tasks[input.index] = types.NewTask(globalId, msg.Gas(), &msg, bHash, input.tx.Hash())
	})

	for i, tx := range txs {
		wg.Add(1)
		pool.Invoke(struct {
			index int
			tx    types2.Transaction
		}{index: i, tx: tx})
	}
	wg.Wait()
	pool.Release()

	return tasks
}
