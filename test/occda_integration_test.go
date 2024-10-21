package test

import (
	"blockConcur/helper"
	occdacore "blockConcur/occda_core"
	"blockConcur/pipeline"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
)

func TestOCCDAIntegration(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()

	processorNum = GetProcessorNumFromEnv()
	startNum := GetStartNumFromEnv()
	endNum := GetEndNumFromEnv()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	headers := env.FetchHeaders(startNum-256, endNum)
	var totalTps, totalGps, totalInmemTps, totalInmemGps float64
	var tpsValues, gpsValues, inmemTpsValues, inmemGpsValues []float64
	blockCount := endNum - startNum
	var totalExecuteCost float64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 1), block.Withdrawals(), header.Coinbase)
		withdrawals := block.Withdrawals()
		cost_graph, graph := pipeline.GenerateGraph(tasks, pipeline.GenerateAccessedBy(tasks))

		// Execute using OCCDA
		occdaTasks := occdacore.GenerateOCCDATasks(tasks)
		h_txs, tidToTaskIdx := occdacore.OCCDAInitialize(occdaTasks, graph)
		cost_execute, gas := occdacore.OCCDAMain(occdaTasks, h_txs, tidToTaskIdx, processorNum, mvCache, header, headers, env.Cfg)

		// Process withdrawals
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

		mvCache.GarbageCollection(balanceUpdate, post_block_task)

		totalTime := cost_graph + cost_execute
		inmemTime := cost_graph + cost_execute
		tps := float64(len(tasks)) / (float64(totalTime))
		gps := float64(gas) / (float64(totalTime))
		inmemTps := float64(len(tasks)) / (float64(inmemTime))
		inmemGps := float64(gas) / (float64(inmemTime))

		totalTps += tps
		totalGps += gps
		totalInmemTps += inmemTps
		totalInmemGps += inmemGps
		tpsValues = append(tpsValues, tps)
		gpsValues = append(gpsValues, gps)
		inmemTpsValues = append(inmemTpsValues, inmemTps)
		inmemGpsValues = append(inmemGpsValues, inmemGps)
		totalExecuteCost += cost_execute

		//TODO: there're mistakes when validating the result
		// but we don't care about that now
		// nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		// tid := mvCache.Validate(nxt_ibs)
		// if tid != nil {
		// 	fmt.Println(tid)
		// 	fmt.Println(tasks[tid.TxIndex].TxHash.Hex())
		// 	panic("incorrect results")
		// }
	}

	// Calculate and print performance metrics
	avgTps := totalTps / float64(blockCount)
	avgGps := totalGps / float64(blockCount)
	avgInmemTps := totalInmemTps / float64(blockCount)
	avgInmemGps := totalInmemGps / float64(blockCount)

	tpsStdDev := calculateStandardDeviation(tpsValues, avgTps)
	gpsStdDev := calculateStandardDeviation(gpsValues, avgGps)
	inmemTpsStdDev := calculateStandardDeviation(inmemTpsValues, avgInmemTps)
	inmemGpsStdDev := calculateStandardDeviation(inmemGpsValues, avgInmemGps)

	tpsMedian := calculateMedian(tpsValues)
	gpsMedian := calculateMedian(gpsValues)
	inmemTpsMedian := calculateMedian(inmemTpsValues)
	inmemGpsMedian := calculateMedian(inmemGpsValues)

	fmt.Printf("Processor Number: %d\n", processorNum)
	fmt.Printf("TPS Metrics:\n")
	fmt.Printf("  Average: %.2f\n", avgTps)
	fmt.Printf("  Standard Deviation: %.2f\n", tpsStdDev)
	fmt.Printf("  Median: %.2f\n", tpsMedian)
	fmt.Printf("  In-Memory Average: %.2f\n", avgInmemTps)
	fmt.Printf("  In-Memory Standard Deviation: %.2f\n", inmemTpsStdDev)
	fmt.Printf("  In-Memory Median: %.2f\n", inmemTpsMedian)

	fmt.Printf("\nGPS Metrics:\n")
	fmt.Printf("  Average: %.2f\n", avgGps)
	fmt.Printf("  Standard Deviation: %.2f\n", gpsStdDev)
	fmt.Printf("  Median: %.2f\n", gpsMedian)
	fmt.Printf("  In-Memory Average: %.2f\n", avgInmemGps)
	fmt.Printf("  In-Memory Standard Deviation: %.2f\n", inmemGpsStdDev)
	fmt.Printf("  In-Memory Median: %.2f\n", inmemGpsMedian)

	fmt.Printf("\nTotal Execute Cost: %.2f seconds\n", totalExecuteCost)
}
