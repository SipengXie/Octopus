package test

import (
	"fmt"
	"octopus/eutils"
	"octopus/evm/vm"
	"octopus/helper"
	"octopus/rwset"
	"octopus/state"
	"octopus/types"
	"testing"

	core "octopus/evm"

	"github.com/ledgerwatch/erigon/params"
	"golang.org/x/exp/rand"
)

func TestDoubleExecution(t *testing.T) {
	env := helper.PrepareEnv()

	groupSize := 100
	groupCount := 20
	totalBlocks := int(endNum - startNum)

	groups := make([]uint64, groupCount)
	for i := 0; i < groupCount; i++ {
		groups[i] = startNum + uint64(rand.Intn(totalBlocks-groupSize))
	}

	results := make(chan string, groupCount)

	for i := 0; i < groupCount; i++ {
		go func(group int) {
			dbTx, err := env.DB.BeginRo(env.Ctx)
			if err != nil {
				results <- fmt.Sprintf("Group %d: Failed to create database transaction: %v", group+1, err)
				return
			}
			defer dbTx.Rollback()

			startBlock := groups[group]
			endBlock := startBlock + uint64(groupSize)

			totalInaccurateTxs := 0
			totalDifferentRwSets := 0

			for blockNum := startBlock; blockNum < endBlock; blockNum++ {
				block, header := env.GetBlockAndHeader(blockNum)
				ibs1 := env.GetIBS(blockNum, dbTx)
				ibs2 := env.GetIBS(blockNum, dbTx)
				headers := env.FetchHeaders(blockNum-256, blockNum)

				accurateTasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs1, convertNum)
				predictTasks := helper.GeneratePredictRwSets(block.Transactions(), header, headers, ibs2, convertNum)

				inaccurateTxs := findInaccurateTxs(accurateTasks, predictTasks)

				ibs := env.GetIBS(blockNum, dbTx)
				cfg := params.MainnetChainConfig
				tasks := helper.ConvertTxToTasks(block.Transactions(), header, convertNum)
				execCtx := eutils.NewExecContext(header, headers, cfg, false)
				execState := state.NewForRwSetGen(ibs, header.Coinbase, false, 8192)
				execCtx.ExecState = execState
				deferedTasks := make(types.Tasks, 0)

				for _, task := range tasks {
					newRwSet := rwset.NewRwSet()
					execCtx.SetTask(task, newRwSet)
					evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
					_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true, false)
					if err != nil {
						task.RwSet = newRwSet
						deferedTasks = append(deferedTasks, task)
						continue
					}
					if !contains(inaccurateTxs, task.Tid.TxIndex) {
						execState.Commit()
					} else {
						task.RwSet = newRwSet
						deferedTasks = append(deferedTasks, task)
					}
				}

				differentRwSetCount := 0
				for _, task := range deferedTasks {
					newRwSet := task.RwSet
					newRwSet2 := rwset.NewRwSet()
					task.RwSet = nil
					execCtx.SetTask(task, newRwSet2)
					evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
					_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true, false)
					if err != nil {
						differentRwSetCount++
						// There might be situations that the reordered transactions will not produce enought gas
						// so that the coinbase-transfer transaction won't succeed.
						fmt.Printf("Error: %v, Transaction hash: %v, Block number: %v\n", err, task.TxHash, blockNum)
						continue
					}

					if !compareRwSets(newRwSet, newRwSet2) {
						differentRwSetCount++
					}

					execState.Commit()
				}

				totalInaccurateTxs += len(inaccurateTxs)
				totalDifferentRwSets += differentRwSetCount
			}

			differentRatio := float64(totalDifferentRwSets) / float64(totalInaccurateTxs)
			results <- fmt.Sprintf("Group %d (Blocks %d - %d): Total inaccurate transactions = %d, Total transactions with different RwSets after two executions = %d, Ratio = %.2f%%",
				group+1, startBlock, endBlock-1, totalInaccurateTxs, totalDifferentRwSets, differentRatio*100)
		}(i)
	}

	for i := 0; i < groupCount; i++ {
		t.Log(<-results)
	}
}

func findInaccurateTxs(accurateTasks, predictTasks []*types.Task) []int {
	inaccurateTxs := []int{}
	for i := range accurateTasks {
		if !compareRwSets(accurateTasks[i].RwSet, predictTasks[i].RwSet) {
			inaccurateTxs = append(inaccurateTxs, i)
		}
	}
	return inaccurateTxs
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func TestDoubleExecutionSingleBlock(t *testing.T) {
	env := helper.PrepareEnv()

	// Select a specific block number for testing
	blockNum := uint64(18596427) // You can modify this value as needed

	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatalf("Failed to create database transaction: %v", err)
	}
	defer dbTx.Rollback()

	ibs1 := env.GetIBS(blockNum, dbTx)
	ibs2 := env.GetIBS(blockNum, dbTx)
	block, header := env.GetBlockAndHeader(blockNum)
	txs := block.Transactions()
	headers := env.FetchHeaders(blockNum-256, blockNum)

	accurateTasks := helper.GenerateAccurateRwSets(txs, header, headers, ibs1, convertNum)
	predictTasks := helper.GeneratePredictRwSets(txs, header, headers, ibs2, convertNum)

	inaccurateTxs := findInaccurateTxs(accurateTasks, predictTasks)

	ibs := env.GetIBS(blockNum, dbTx)
	cfg := params.MainnetChainConfig
	tasks := helper.ConvertTxToTasks(block.Transactions(), header, convertNum)
	execCtx := eutils.NewExecContext(header, headers, cfg, false)
	execState := state.NewForRwSetGen(ibs, header.Coinbase, false, 8192)
	execCtx.ExecState = execState
	deferedTasks := make(types.Tasks, 0)

	for _, task := range tasks {
		newRwSet := rwset.NewRwSet()
		execCtx.SetTask(task, newRwSet)
		evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true, false)
		if err != nil {
			task.RwSet = newRwSet
			deferedTasks = append(deferedTasks, task)
			continue
		}
		if !contains(inaccurateTxs, task.Tid.TxIndex) {
			execState.Commit()
		} else {
			task.RwSet = newRwSet
			deferedTasks = append(deferedTasks, task)
		}
	}

	differentRwSetCount := 0
	for _, task := range deferedTasks {
		newRwSet := task.RwSet
		newRwSet2 := rwset.NewRwSet()
		task.RwSet = nil
		execCtx.SetTask(task, newRwSet2)
		evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
		_, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true, false)
		if err != nil {
			differentRwSetCount++
			// There might be situations that the reordered transactions will not produce enought gas
			// so that the coinbase-transfer transaction won't succeed.
			fmt.Printf("Error: %v, Transaction hash: %v, Block number: %v\n", err, task.TxHash, blockNum)
			continue
		}

		if !compareRwSets(newRwSet, newRwSet2) {
			differentRwSetCount++
		}

		execState.Commit()
	}

}
