package test

import (
	"blockConcur/helper"
	"blockConcur/pipeline"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"sync"
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
	processorNum := GetProcessorNumFromEnv()
	startNum := GetStartNumFromEnv()
	endNum := GetEndNumFromEnv()
	headers := env.FetchHeaders(startNum-256, endNum)
	mvCache := state.NewMvCache(env.GetIBS(uint64(startNum), dbTx), cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	var totalCriticalPathLen uint64
	var totalMakespanHEFT, totalMakespanHESI, totalMakespanLOBA, totalMakespanCPTL, totalMakespanCPOP, totalMakespanPEFT uint64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 1), block.Withdrawals(), header.Coinbase)

		_, rwAccessedBy := pipeline.Prefetch(tasks, post_block_task, fetchPool, ivPool)
		_, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		totalCriticalPathLen += graph.CriticalPathLen

		var wg sync.WaitGroup
		wg.Add(6)
		var makespanHEFT, makespanHESI, makespanLOBA, makespanCPTL, makespanCPOP, makespanPEFT uint64

		go func() {
			defer wg.Done()
			_, _, makespanHEFT, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.HEFT)
		}()

		go func() {
			defer wg.Done()
			_, _, makespanHESI, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.HESI)
		}()

		go func() {
			defer wg.Done()
			_, _, makespanLOBA, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.LOBA)
		}()

		go func() {
			defer wg.Done()
			_, _, makespanCPTL, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.CPTL)
		}()

		go func() {
			defer wg.Done()
			_, _, makespanCPOP, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.CPOP)
		}()

		go func() {
			defer wg.Done()
			_, _, makespanPEFT, _ = pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.PEFT)
		}()

		wg.Wait()

		totalMakespanHEFT += makespanHEFT
		totalMakespanHESI += makespanHESI
		totalMakespanLOBA += makespanLOBA
		totalMakespanCPTL += makespanCPTL
		totalMakespanCPOP += makespanCPOP
		totalMakespanPEFT += makespanPEFT
	}

	slrHEFT := float64(totalMakespanHEFT) / float64(totalCriticalPathLen)
	slrHESI := float64(totalMakespanHESI) / float64(totalCriticalPathLen)
	slrLOBA := float64(totalMakespanLOBA) / float64(totalCriticalPathLen)
	slrCPTL := float64(totalMakespanCPTL) / float64(totalCriticalPathLen)
	slrCPOP := float64(totalMakespanCPOP) / float64(totalCriticalPathLen)
	slrPEFT := float64(totalMakespanPEFT) / float64(totalCriticalPathLen)

	extraTimeHEFT := totalMakespanHEFT - totalCriticalPathLen
	extraTimeHESI := totalMakespanHESI - totalCriticalPathLen
	extraTimeLOBA := totalMakespanLOBA - totalCriticalPathLen
	extraTimeCPTL := totalMakespanCPTL - totalCriticalPathLen
	extraTimeCPOP := totalMakespanCPOP - totalCriticalPathLen
	extraTimePEFT := totalMakespanPEFT - totalCriticalPathLen

	fmt.Printf("Number of Processors: %d\n", processorNum)
	fmt.Printf("Overall SLR - HEFT: %.2f%%, HESI: %.2f%%, LOBA: %.2f%%, CPTL: %.2f%%, CPOP: %.2f%%, PEFT: %.2f%%\n", slrHEFT*100, slrHESI*100, slrLOBA*100, slrCPTL*100, slrCPOP*100, slrPEFT*100)
	fmt.Printf("Extra execution gas compared to optimal - HEFT: %d, HESI: %d, LOBA: %d, CPTL: %d, CPOP: %d, PEFT: %d\n", extraTimeHEFT, extraTimeHESI, extraTimeLOBA, extraTimeCPTL, extraTimeCPOP, extraTimePEFT)
}
