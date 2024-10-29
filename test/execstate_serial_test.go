package test

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/helper"
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/params"
)

// This file will test whether we can correctly execute transactions serially using mvcache as the underlying layer
// The tasks built from these transactions will not contain read_version and write_version, so no versions will be generated

func Test_Serial_Exec_ColdState(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()

	cfg := params.MainnetChainConfig
	startNum := GetStartNumFromEnv()
	endNum := GetEndNumFromEnv()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	headers := env.FetchHeaders(startNum-256, endNum)

	// Track metrics across all blocks
	var tpsValues []float64
	var gpsValues []float64
	var totalTxs uint64
	var totalGasUsed uint64
	var totalDuration time.Duration

	var ret uint256.Int
	slot := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000b3")
	ibs.GetState(common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02"), &slot, &ret)
	fmt.Println(ret, ret.Hex())

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		txs := block.Transactions()
		withdrawals := block.Withdrawals()
		tasks := helper.ConvertTxToTasks(txs, header, convertNum)
		execCtx := eutils.NewExecContext(header, headers, cfg, false)
		execState := state.NewForRun(mvCache, header.Coinbase, early_abort)
		execCtx.ExecState = execState

		startTime := time.Now()
		totalGas := uint64(0)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 5), withdrawals, header.Coinbase)
		balanceUpdate := make(map[common.Address]*uint256.Int)
		for _, withdrawal := range withdrawals {
			balance, ok := balanceUpdate[withdrawal.Address]
			if !ok {
				balance = uint256.NewInt(0)
			}
			factor := new(uint256.Int).SetUint64(1000000000)
			amount := new(uint256.Int).Mul(new(uint256.Int).SetUint64(withdrawal.Amount), factor)
			balance.Add(balance, amount)
			balanceUpdate[withdrawal.Address] = balance
		}

		for _, task := range tasks {
			newRwSet := rwset.NewRwSet()
			execCtx.SetTask(task, newRwSet)
			// evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
			var tracer vm.EVMLogger
			var evm *vm.EVM
			if task.Tid.BlockNumber == 19672814 && task.Tid.TxIndex == 10 {
				fmt.Println(task.TxHash)
				tracer = helper.NewStructLogger(&helper.LogConfig{})
				evm = vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{Debug: true, Tracer: tracer})
			} else {
				evm = vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
			}

			result, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
			if err != nil {
				panic(fmt.Sprintf("Error: %v, Transaction hash: %v", err, task.TxHash))
			}
			task.RwSet = newRwSet
			execState.Commit()

			if tracer != nil {
				if structLogs, ok := tracer.(*helper.StructLogger); ok {
					structLogs.Flush(task.TxHash)
					panic("debug")
				}
			}

			totalGas += result.UsedGas
		}
		mvCache.GarbageCollection(balanceUpdate, post_block_task)

		duration := time.Since(startTime)
		totalDuration += duration
		totalTxs += uint64(len(txs))
		totalGasUsed += totalGas

		// Calculate per-block metrics
		tps := float64(len(txs)) / duration.Seconds()
		gps := float64(totalGas) / duration.Seconds()
		tpsValues = append(tpsValues, tps)
		gpsValues = append(gpsValues, gps)

		nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(nxt_ibs)
		if tid != nil {
			panic(fmt.Sprintf("State validation failed at block %d, tid: %v", blockNum, tid))
		}
	}

	// Calculate overall metrics
	totalTps := float64(totalTxs) / totalDuration.Seconds()
	totalGps := float64(totalGasUsed) / totalDuration.Seconds()

	// Calculate standard deviation and median
	tpsStdDev := calculateStandardDeviation(tpsValues, totalTps)
	gpsStdDev := calculateStandardDeviation(gpsValues, totalGps)
	tpsMedian := calculateMedian(tpsValues)
	gpsMedian := calculateMedian(gpsValues)

	fmt.Printf("Total TPS: %.2f, TPS StdDev: %.2f, TPS Median: %.2f\n", totalTps, tpsStdDev, tpsMedian)
	fmt.Printf("Total GPS: %.2f, GPS StdDev: %.2f, GPS Median: %.2f\n", totalGps, gpsStdDev, gpsMedian)
}
