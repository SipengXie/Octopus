package test

import (
	"fmt"
	"octopus/helper"
	"octopus/pipeline"
	"octopus/types"
	"testing"

	"github.com/ledgerwatch/erigon-lib/kv"
)

func TestTreeListSchedule(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()

	taskCounts := []int{
		200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000,
		2200, 2400, 2600, 2800, 3000, 3200, 3400, 3600, 3800, 4000,
		4200, 4400, 4600, 4800, 5000, 5200, 5400, 5600, 5800, 6000,
		6200, 6400, 6600, 6800, 7000, 7200, 7400, 7600, 7800, 8000,
		8200, 8400, 8600, 8800, 9000, 9200, 9400, 9600, 9800, 10000,
	}
	processorNum := GetProcessorNumFromEnv()

	for _, taskCount := range taskCounts {
		fmt.Printf("Testing with %d tasks:\n", taskCount)

		// Collect tasks
		tasks := collectTasks(env, dbTx, taskCount)

		// Generate graph
		rwAccessedBy := pipeline.GenerateAccessedBy(tasks)
		_, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)

		// Schedule using ProcessorList
		listTime, _, _, _ := pipeline.Schedule(graph, false, processorNum, pipeline.octopus)

		treeTime, _, _, _ := pipeline.Schedule(graph, true, processorNum, pipeline.octopus)

		fmt.Printf("  ProcessorList scheduling time: %v\n", listTime)
		fmt.Printf("  ProcessorTree scheduling time: %v\n", treeTime)
	}
}

func collectTasks(env helper.GloablEnv, dbTx kv.Tx, count int) types.Tasks {
	var tasks types.Tasks
	blockNum := startNum

	for len(tasks) < count {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		blockTasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs, convertNum)

		remainingSpace := count - len(tasks)
		if len(blockTasks) <= remainingSpace {
			tasks = append(tasks, blockTasks...)
		} else {
			tasks = append(tasks, blockTasks[:remainingSpace]...)
		}

		blockNum++
	}

	return tasks
}
