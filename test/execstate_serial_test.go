package test

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/helper"
	"blockConcur/rwset"
	"blockConcur/state"
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
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		txs := block.Transactions()
		withdrawals := block.Withdrawals()
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.ConvertTxToTasks(txs, header, convertNum)
		execCtx := eutils.NewExecContext(header, headers, cfg, false)
		execState := state.NewForRun(mvCache, header.Coinbase, early_abort)
		execCtx.ExecState = execState

		startTime := time.Now()
		totalGas := uint64(0)

		balanceUpdate := make(map[common.Address]*uint256.Int)
		for _, withdrawal := range withdrawals {
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

		for _, task := range tasks {
			newRwSet := rwset.NewRwSet()
			execCtx.SetTask(task, newRwSet)
			evm := vm.NewEVM(execCtx.BlockCtx, execCtx.TxCtx, execState, execCtx.ChainCfg, vm.Config{})
			result, err := core.ApplyMessage(evm, task.Msg, new(core.GasPool).AddGas(task.Msg.Gas()).AddBlobGas(task.Msg.BlobGas()), true /* refunds */, false /* gasBailout */)
			if err != nil {
				panic(fmt.Sprintf("Error: %v, Transaction hash: %v", err, task.TxHash))
			}
			task.RwSet = newRwSet
			execState.Commit()

			totalGas += result.UsedGas
		}
		mvCache.GarbageCollection(blockNum, len(tasks), balanceUpdate)

		duration := time.Since(startTime)
		tps := float64(len(txs)) / duration.Seconds()
		gps := float64(totalGas) / duration.Seconds()

		fmt.Printf("Block %d: TPS = %.2f, GPS = %.2f\n", blockNum, tps, gps)

		ibs_bak := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(ibs_bak)
		if tid != nil {
			fmt.Println(tid)
		}
	}
}
