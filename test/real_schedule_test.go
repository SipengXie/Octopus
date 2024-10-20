package test

import (
	"blockConcur/helper"
	"blockConcur/pipeline"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"testing"
)

// This code will test the performance of different scheduling methods on real Ethereum transactions
// We will use scheduler_load_balancing, scheduler_heuristic_simple, and scheduler_aggregator for testing
// Compare their SLR under different processorNum

func TestRealSchedule(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	mvCache := state.NewMvCache(env.GetIBS(uint64(startNum), dbTx), cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	processorNum := GetProcessorNumFromEnv()
	var totalMakespanOCCDA, totalMakespanQUECC, totalMakespanBlkConcur uint64
	var totalCriticalPathLen uint64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 1), block.Withdrawals(), header.Coinbase)

		_, rwAccessedBy := pipeline.Prefetch(tasks, post_block_task, fetchPool, ivPool)
		_, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		totalCriticalPathLen += graph.CriticalPathLen
		_, _, makespanBlkConcur, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.BlkConcur)

		// Using SchedulerHESI
		_, _, makespanOCCDA, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.OCCDA_MOCK)

		// Using SchedulerLOBA
		_, _, makespanQUECC, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.QUECC_MOCK)

		totalMakespanOCCDA += makespanOCCDA
		totalMakespanQUECC += makespanQUECC
		totalMakespanBlkConcur += makespanBlkConcur
	}

	slrOCCDA := float64(totalMakespanOCCDA) / float64(totalCriticalPathLen)
	slrQUECC := float64(totalMakespanQUECC) / float64(totalCriticalPathLen)
	slrBlkConcur := float64(totalMakespanBlkConcur) / float64(totalCriticalPathLen)

	extraTimeOCCDA := totalMakespanOCCDA - totalCriticalPathLen
	extraTimeQUECC := totalMakespanQUECC - totalCriticalPathLen
	extraTimeBlkConcur := totalMakespanBlkConcur - totalCriticalPathLen

	fmt.Printf("Number of Processors: %d\n", processorNum)
	fmt.Printf("Overall SLR - OCCDA: %.2f%%, QUECC: %.2f%%, BlkConcur: %.2f%%\n", slrOCCDA*100, slrQUECC*100, slrBlkConcur*100)
	fmt.Printf("Extra execution gas compared to optimal - OCCDA: %d, QUECC: %d, BlkConcur: %d\n", extraTimeOCCDA, extraTimeQUECC, extraTimeBlkConcur)
}
