package helper

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"fmt"

	"github.com/ledgerwatch/erigon-lib/common"
	types2 "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

// Generate Accurate Read-write sets,
func GenerateAccurateRwSets(txs types2.Transactions, header *types2.Header, headers []*types2.Header, ibs *state.IntraBlockState, worker_num int) types.Tasks {
	cfg := params.MainnetChainConfig
	tasks := ConvertTxToTasks(txs, header, worker_num)
	// 用于串行执行
	execCtx := eutils.NewExecContext(header, headers, cfg, false)
	execState := state.NewForRwSetGen(ibs, header.Coinbase, false, 8192)
	execCtx.ExecState = execState
	for _, task := range tasks {
		newRwSet := rwset.NewRwSet()
		execCtx.SetTask(task, newRwSet)
		evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})

		/* This code section is used for debugging
		var tracer vm.EVMLogger
		var evm *vm.EVM
		if task.TxHash == common.HexToHash("0x83d6a34cf13f93bc418ceb5ced9b61f640a3e936fbd98f6d8c6d4896ab70d12b") {
			tracer = NewStructLogger(&LogConfig{})
			evm = vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{Debug: true, Tracer: tracer})
		} else {
			evm = vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
		}
		*/

		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err != nil {
			panic(fmt.Sprintf("error: %v, txHash:%v", err, task.TxHash))
		}

		/* This code section is used for debugging
		if task.TxHash == common.HexToHash("0x83d6a34cf13f93bc418ceb5ced9b61f640a3e936fbd98f6d8c6d4896ab70d12b") {
			if structLogs, ok := tracer.(*StructLogger); ok {
				structLogs.Flush(task.TxHash)
			}
		}
		*/

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
			// if it happens, we can generate some basic rwset
			// we could skip it, or provide some basic information
			fmt.Printf("error: %v, txHash:%v\n", err, task.TxHash)
			newRwSet = rwset.NewRwSet()
			is_transfer := !task.Msg.Value().IsZero()
			is_coinbase := task.Msg.From() == header.Coinbase
			is_call := task.Msg.To() != nil && len(execState.GetCode(*task.Msg.To())) > 0
			var to common.Address
			if task.Msg.To() != nil {
				to = *task.Msg.To()
			}
			newRwSet.BasicRwSet(task.Msg.From(), to, is_transfer, is_coinbase, is_call)
		}
		task.RwSet = newRwSet
		output = append(output, task)
	}
	return output
}
