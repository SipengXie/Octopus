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

func TestPipeline(t *testing.T) {
	// Prepare the environment
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()

	// shared resources
	processorNum := GetProcessorNumFromEnv()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	wg := &sync.WaitGroup{}
	taskChan := make(chan *pipeline.TaskMessage, 1024)
	buildGraphChan := make(chan *pipeline.BuildGraphMessage, 1024)
	graphChan := make(chan *pipeline.GraphMessage, 1024)
	scheduleChan := make(chan *pipeline.ScheduleMessage, 1024)

	prefetcher := pipeline.NewPrefetcher(mvCache, wg, fetchPoolSize, ivPoolSize, taskChan, buildGraphChan)
	graphBuilder := pipeline.NewGraphBuilder(wg, buildGraphChan, graphChan)
	scheduler := pipeline.NewScheduler(processorNum, false, wg, graphChan, scheduleChan)
	executor := pipeline.NewExecutor(mvCache, env.Cfg, early_abort, wg, scheduleChan)

	// Start the pipeline components
	wg.Add(4)
	go prefetcher.Run()
	go graphBuilder.Run()
	go scheduler.Run()
	go executor.Run()

	// Collect and send tasks
	dbTx2, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx2.Rollback()
	totalTxs := 0
	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx2)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		totalTxs += len(tasks)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 1), block.Withdrawals(), header.Coinbase)
		taskMessage := &pipeline.TaskMessage{
			Flag:      pipeline.START,
			Tasks:     tasks,
			PostBlock: post_block_task,
			Header:    header,
			Headers:   headers,
			Withdraws: block.Withdrawals(),
		}
		taskChan <- taskMessage
	}

	// Send END signal and close channels
	taskChan <- &pipeline.TaskMessage{Flag: pipeline.END}
	close(taskChan)
	wg.Wait()

	fmt.Printf("Total transactions processed: %d\n", totalTxs)

	nxt_ibs := env.GetIBS(uint64(endNum), dbTx)
	tid := mvCache.Validate(nxt_ibs)
	if tid != nil {
		fmt.Println(tid)
		panic("incorrect results")
	}

}
