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

func TestSingleBlock(t *testing.T) {
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
	var totalTps, totalGps, totalInmemTps, totalInmemGps float64
	var tpsValues, gpsValues, inmemTpsValues, inmemGpsValues []float64
	blockCount := endNum - startNum
	var totalExecuteCost float64

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)

		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 5), block.Withdrawals(), header.Coinbase)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, post_block_task, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.BlkConcur)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), post_block_task, header, headers, env.Cfg, early_abort, mvCache)

		totalTime := cost_prefetch + cost_graph + cost_schedule + cost_execute
		inmemTime := cost_graph + cost_schedule + cost_execute
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

		nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(nxt_ibs)
		if tid != nil {
			fmt.Println(blockNum)
			fmt.Println(tid)
			fmt.Println(tasks[tid.TxIndex].TxHash.Hex())
			panic("incorrect results")
		}
	}

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

func TestSingleBlockPredict(t *testing.T) {
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
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)
	var totalTps, totalGps, totalInmemTps, totalInmemGps float64
	var tpsValues, gpsValues, inmemTpsValues, inmemGpsValues []float64
	blockCount := endNum - startNum

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)

		tasks := helper.GeneratePredictRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)
		post_block_task := types.NewPostBlockTask(utils.NewID(uint64(blockNum), len(tasks), 5), block.Withdrawals(), header.Coinbase)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, post_block_task, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum, pipeline.BlkConcur)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), post_block_task, header, headers, env.Cfg, early_abort, mvCache)

		totalTime := cost_prefetch + cost_graph + cost_schedule + cost_execute
		inmemTime := cost_graph + cost_schedule + cost_execute
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
	}

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
}
