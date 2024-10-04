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
	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, env.Headers, ibs_bak, convertNum)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		fmt.Printf("BlockNum: %d, Fetch Cost: %.2f ms\n", blockNum, cost_prefetch*1000)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		fmt.Printf("BlockNum: %d, Graph Cost: %.2f ms\n", blockNum, cost_graph*1000)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		fmt.Printf("BlockNum: %d, Schedule Cost: %.2f ms\n", blockNum, cost_schedule*1000)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, env.Headers, env.Cfg, early_abort, mvCache)
		fmt.Printf("BlockNum: %d, Execute Cost: %.2f ms\n", blockNum, cost_execute*1000)

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
		fmt.Printf("BlockNum: %d, TPS: %.2f, GPS: %.2f, ITPS: %.2f, IGPS: %.2f\n", blockNum, tps, gps, itps, igps)
	}
}
