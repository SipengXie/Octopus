package helper

import (
	"fmt"
	"octopus/eutils"
	core "octopus/evm"
	"octopus/evm/vm"
	"octopus/rwset"
	"octopus/state"
	"octopus/types"
	"octopus/utils"

	"github.com/ledgerwatch/erigon-lib/common"
	types3 "github.com/ledgerwatch/erigon-lib/types"
	types2 "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
)

// Generate Accurate Read-write sets,
func GenerateAccurateRwSets(txs types2.Transactions, header *types2.Header, headers []*types2.Header, ibs *state.IntraBlockState, worker_num int) types.Tasks {
	cfg := params.MainnetChainConfig
	tasks := ConvertTxToTasks(txs, header, worker_num)
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

		res, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
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
		// if len(task.Msg.AccessList()) > 0 {
		// 	mergeAccessList(task.Msg.AccessList(), newRwSet)
		// }
		task.Cost = res.UsedGas
		task.RwSet = newRwSet
		execState.Commit()
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

		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, true /* gasBailout */)
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
		if len(task.Msg.AccessList()) > 0 {
			mergeAccessList(task.Msg.AccessList(), newRwSet)
		}
		task.RwSet = newRwSet
		output = append(output, task)
	}
	return output
}

func mergeAccessList(accessList types3.AccessList, rwSet *rwset.RwSet) {

	for _, access := range accessList {
		address := access.Address
		for _, storageKey := range access.StorageKeys {
			rwSet.AddReadSet(address, storageKey)
			rwSet.AddWriteSet(address, storageKey)
		}
		rwSet.AddReadSet(address, utils.BALANCE)
		rwSet.AddReadSet(address, utils.NONCE)
		rwSet.AddReadSet(address, utils.CODE)
		rwSet.AddReadSet(address, utils.CODEHASH)
		rwSet.AddReadSet(address, utils.EXIST)

		rwSet.AddWriteSet(address, utils.BALANCE)
		rwSet.AddWriteSet(address, utils.NONCE)
		rwSet.AddWriteSet(address, utils.CODE)
		rwSet.AddWriteSet(address, utils.CODEHASH)
		rwSet.AddWriteSet(address, utils.EXIST)
	}

}
