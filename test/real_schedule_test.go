package test

import (
	"blockConcur/helper"
	"blockConcur/pipeline"
	"blockConcur/schedule"
	"blockConcur/state"
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

	var totalMakespanOCCDA, totalMakespanQUECC, totalMakespanBlkConcur uint64
	var totalCriticalPathLen uint64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		_, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		_, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		totalCriticalPathLen += graph.CriticalPathLen
		_, _, makespanBlkConcur, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)

		// Using SchedulerHESI
		processorsHESI := make(schedule.Processors, processorNum)
		for i := 0; i < processorNum; i++ {
			processorsHESI[i] = schedule.NewProcessorSimple()
		}
		schedulerHESI := schedule.NewSchedulerHESI(graph, processorsHESI)
		schedulerHESI.Schedule()
		makespanOCCDA := schedulerHESI.Makespan()

		// Using SchedulerLOBA
		processorsLOBA := make(schedule.Processors, processorNum)
		for i := 0; i < processorNum; i++ {
			processorsLOBA[i] = schedule.NewProcessorSimple()
		}
		schedulerLOBA := schedule.NewSchedulerLOBA(graph, processorsLOBA)
		schedulerLOBA.Schedule()
		makespanQUECC := schedulerLOBA.Makespan()

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
	fmt.Printf("Overall SLR - OCCDA: %.2f, QUECC: %.2f, BlkConcur: %.2f\n", slrOCCDA, slrQUECC, slrBlkConcur)
	fmt.Printf("Extra execution gas compared to optimal - OCCDA: %d, QUECC: %d, BlkConcur: %d\n", extraTimeOCCDA, extraTimeQUECC, extraTimeBlkConcur)
}
