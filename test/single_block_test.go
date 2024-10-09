package test

import (
	"blockConcur/helper"
	"blockConcur/pipeline"
	"blockConcur/state"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

func GetProcessorNumFromEnv() int {
	processorNumStr := os.Getenv("PROCESSOR_NUM")
	if processorNumStr != "" {
		if num, err := strconv.Atoi(processorNumStr); err == nil {
			return num
		}
	}
	return processorNum
}

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

	var totalTps, totalGps, totalInmemTps, totalInmemGps float64
	var tpsValues, gpsValues, inmemTpsValues, inmemGpsValues []float64
	blockCount := endNum - startNum

	processorNum = GetProcessorNumFromEnv()

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, headers, env.Cfg, early_abort, mvCache)

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

		nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(nxt_ibs)
		if tid != nil {
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
}

// Helper function to calculate standard deviation
func calculateStandardDeviation(values []float64, mean float64) float64 {
	var sum float64
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	variance := sum / float64(len(values))
	return math.Sqrt(variance)
}

// Helper function to calculate median
func calculateMedian(values []float64) float64 {
	sort.Float64s(values)
	length := len(values)
	if length%2 == 0 {
		return (values[length/2-1] + values[length/2]) / 2
	}
	return values[length/2]
}

func TestSingleBlockPredict(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()
	ibs := env.GetIBS(uint64(startNum), dbTx)
	mvCache := state.NewMvCache(ibs, cacheSize)
	fetchPool, ivPool := pipeline.GeneratePools(mvCache, fetchPoolSize, ivPoolSize)

	var totalTps, totalGps, totalInmemTps, totalInmemGps float64
	var tpsValues, gpsValues, inmemTpsValues, inmemGpsValues []float64
	blockCount := endNum - startNum

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs_bak := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)
		tasks := helper.GeneratePredictRwSets(block.Transactions(), header, headers, ibs_bak, convertNum)

		cost_prefetch, rwAccessedBy := pipeline.Prefetch(tasks, fetchPool, ivPool)
		cost_graph, graph := pipeline.GenerateGraph(tasks, rwAccessedBy)
		cost_schedule, processors, _, _ := pipeline.Schedule(graph, use_tree(len(tasks)), processorNum)
		cost_execute, gas := pipeline.Execute(processors, block.Withdrawals(), header, headers, env.Cfg, early_abort, mvCache)

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

		nxt_ibs := env.GetIBS(uint64(blockNum+1), dbTx)
		tid := mvCache.Validate(nxt_ibs)
		if tid != nil {
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
}
