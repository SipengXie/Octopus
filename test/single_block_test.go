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
		fmt.Printf("BlockNum: %d, Fetch Cost: %d ms\n", blockNum, cost_prefetch)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		fmt.Printf("BlockNum: %d, Graph Cost: %d ms\n", blockNum, cost_graph)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		fmt.Printf("BlockNum: %d, Schedule Cost: %d ms\n", blockNum, cost_schedule)
		cost_execute, gas := pipeline.Execute(processors, header, env.Headers, env.Cfg, early_abort, mvCache)
		fmt.Printf("BlockNum: %d, Execute Cost: %d ms\n", blockNum, cost_execute)

		totalTime := cost_prefetch + cost_graph + cost_schedule + cost_execute
		tps := float64(len(tasks)) / (float64(totalTime) / 1000)
		gps := float64(gas) / (float64(totalTime) / 1000)
		itps := float64(len(tasks)) / (float64(cost_execute) / 1000)
		igps := float64(gas) / (float64(cost_execute) / 1000)
		fmt.Printf("BlockNum: %d, TPS: %.2f, GPS: %.2f, ITPS: %.2f, IGPS: %.2f\n", blockNum, tps, gps, itps, igps)
	}
}
