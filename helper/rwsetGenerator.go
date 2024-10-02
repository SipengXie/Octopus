package helper

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"fmt"

	types2 "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

// Generate Accurate Read-write sets,
func GenerateAccurateRwSets(txs types2.Transactions, header *types2.Header, headers []*types2.Header, ibs *state.IntraBlockState, worker_num int) types.Tasks {
	cfg := params.MainnetChainConfig
	tasks := ConvertTxToTasks(txs, header, worker_num)
	// used for serial execution
	execCtx := eutils.NewExecContext(header, headers, cfg, false)
	execState := state.NewForRwSetGen(ibs, header.Coinbase, false, 8192)
	execCtx.ExecState = execState
	evm := vm.NewEVM(execCtx.BlockCtx, evmtypes.TxContext{}, execState, execCtx.ChainCfg, vm.Config{})
	for _, task := range tasks {
		newRwSet := rwset.NewRwSet()
		execCtx.SetTask(task, newRwSet)
		evm.TxContext = execCtx.TxCtx
		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err != nil {
			// we have dealt with the coinbase issue
			// panic if the issue happens again
			panic(fmt.Sprintf("error: %v, txHash:%v", err, task.TxHash))
			// when formally use, we should ignore the error and
			// continue
		}
		execState.Commit()
		task.RwSet = newRwSet
	}
	return tasks
}

func GeneratePredictRwSets(txs types2.Transactions, header *types2.Header, headers []*types2.Header, ibs *state.IntraBlockState, worker_num int) types.Tasks {
	cfg := params.MainnetChainConfig
	tasks := ConvertTxToTasks(txs, header, worker_num)
	output := make(types.Tasks, 0)
	execCtx := eutils.NewExecContext(header, headers, cfg, false)

	for _, task := range tasks {
		task.Msg.SetCheckNonce(false)
		ctx := core.NewEVMTxContext(task.Msg)
		ctx.TxHash = task.TxHash

		execState := state.NewForRwSetGen(ibs, header.Coinbase, false, 8192)
		newRwSet := rwset.NewRwSet()
		execState.SetTxContext(task, newRwSet)

		evm := vm.NewEVM(execCtx.BlockCtx, ctx, execState, execCtx.ChainCfg, vm.Config{})

		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err != nil {
			// some transaction may not be predicted
			// if it happens, we can ignore it
			fmt.Printf("error: %v, txHash:%v\n", err, task.TxHash)
			continue
		}
		output = append(output, task)
		task.RwSet = newRwSet
	}

	// the output is ordered by utils.ID
	return output
}

// func SerialExecute(txs types2.Transactions, header *types2.Header, execCtx *types.ExecContext, coldData *state.IntraBlockState) (cost, tps, gps float64, total_gas uint64) {
// 	cfg := params.MainnetChainConfig
// 	execState := state.NewExecState(coldData, nil, header.Coinbase)
// 	evm := vm.NewEVM(execCtx.BlockCtx, evmtypes.TxContext{}, execState, execCtx.ChainCfg, vm.Config{})
// 	rules := evm.ChainRules()
// 	total_gas = uint64(0)
// 	st := time.Now()
// 	for id, tx := range txs {
// 		msg, _ := tx.AsMessage(*types2.LatestSigner(cfg), header.BaseFee, rules)
// 		ctx := core.NewEVMTxContext(msg)
// 		ctx.TxHash = tx.Hash()
// 		evm.TxContext = ctx
// 		execState.SetTxContext(tx.Hash(), header.Hash(), id)
// 		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
// 		if err != nil {
// 			// we have dealt with the coinbase issue
// 			// panic if the issue happens again
// 			panic(fmt.Sprintf("error: %v, txHash:%v", err, tx.Hash()))
// 			// when formally use, we should ignore the error and
// 			// continue
// 		}
// 		execState.Commit(rules)
// 		total_gas += res.UsedGas
// 	}
// 	cost = time.Since(st).Seconds()
// 	return cost, float64(len(txs)) / cost, float64(total_gas) / cost, total_gas
// }
