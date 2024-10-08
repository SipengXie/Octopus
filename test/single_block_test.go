package test

import (
	"blockConcur/helper"
	"blockConcur/pipeline"
	"blockConcur/state"
	"fmt"
	"testing"
)

func TestSingleBlock(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	methodStats := make(map[string]int)

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		fmt.Printf("Block number: %d, Prefetch cost: %.2f ms\n", blockNum, cost_prefetch*1000)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		fmt.Printf("Block number: %d, Graph generation cost: %.2f ms\n", blockNum, cost_graph*1000)
		cost_schedule, processors, makespan, method := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		fmt.Printf("Block number: %d, Scheduling cost: %.2f ms\n", blockNum, cost_schedule*1000)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, headers, env.Cfg, early_abort, mvCache)
		fmt.Printf("Block number: %d, Execution cost: %.2f ms\n", blockNum, cost_execute*1000)

		methodStats[method.String()]++

		nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(nxt_ibs)
		if tid != nil {
			fmt.Println(tid)
			fmt.Println(tasks[tid.TxIndex].TxHash.Hex())
			break
		}

		totalTime := cost_prefetch + cost_graph + cost_schedule + cost_execute
		tps := float64(len(tasks)) / (float64(totalTime))
		gps := float64(gas) / (float64(totalTime))
		itps := float64(len(tasks)) / (float64(cost_execute))
		igps := float64(gas) / (float64(cost_execute))
		fmt.Printf("Block number: %d, TPS: %.2f, GPS: %.2f, ITPS: %.2f, IGPS: %.2f, SLR: %.2f\n", blockNum, tps, gps, itps, igps, float64(makespan)/float64(graph.CriticalPathLen))
	}

	fmt.Println("Scheduling method statistics:")
	for method, count := range methodStats {
		fmt.Printf("%s: %d\n", method, count)
	}
}

func TestAverageTpsGps(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	mvCache := state.NewMvCache(env.GetIBS(uint64(startNum), dbTx), cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	var totalTx, totalGas, totalCost float64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, headers, env.Cfg, early_abort, mvCache)

		totalCost += cost_prefetch + cost_graph + cost_schedule + cost_execute
		totalTx += float64(len(tasks))
		totalGas += float64(gas)
	}

	avgTps := totalTx / totalCost
	avgGps := totalGas / totalCost

	fmt.Printf("ProcessorNum: %d\n", processorNum)
	fmt.Printf("Average TPS: %.2f\n", avgTps)
	fmt.Printf("Average GPS: %.2f\n", avgGps)
}

func TestInMemTpsGps(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	mvCache := state.NewMvCache(env.GetIBS(uint64(startNum), dbTx), cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	var totalTx, totalGas, totalCost float64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		_, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, headers, env.Cfg, early_abort, mvCache)

		totalCost += cost_graph + cost_schedule + cost_execute
		totalTx += float64(len(tasks))
		totalGas += float64(gas)
	}

	avgTps := totalTx / totalCost
	avgGps := totalGas / totalCost

	fmt.Printf("In-Memory Test Results:\n")
	fmt.Printf("Number of Processors: %d\n", processorNum)
	fmt.Printf("Average TPS: %.2f\n", avgTps)
	fmt.Printf("Average GPS: %.2f\n", avgGps)
}
