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

func TestHitRate(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	processorNum := GetProcessorNumFromEnv()
	startNum := GetStartNumFromEnv()
	endNum := GetEndNumFromEnv()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	headers := env.FetchHeaders(startNum-256, endNum)
	mvCache := state.NewMvCache(ibs, cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)

		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 0), block.Withdrawals(), header.Coinbase)

		_, rwAccessedBy := pipeline.Prefetch(tasks, post_block_task, fetchPool, ivPool)
		_, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		_, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.BlkConcur)
		pipeline.Execute(processors, block.Withdrawals(), post_block_task, header, headers, env.Cfg, early_abort, mvCache)

	}

	fmt.Println(mvCache.GetHitRate())
}
