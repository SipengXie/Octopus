package test

import (
	"fmt"
	"math"
	"octopus/helper"
	"sort"
	"sync"
	"testing"

	"golang.org/x/exp/rand"
)

func sampleAccessListRatio(env *helper.GloablEnv, startBlock, endBlock uint64) (int, int, []float64) {
	totalTxs := 0
	totalAccessListTxs := 0
	ratios := make([]float64, 0)

	for blockNum := startBlock; blockNum < endBlock; blockNum++ {
		block, _ := env.GetBlockAndHeader(blockNum)
		txs := block.Transactions()
		totalTxs += len(txs)

		for _, tx := range txs {
			if len(tx.GetAccessList()) > 0 {
				totalAccessListTxs++
			}
		}
	}

	for blockNum := startBlock; blockNum < endBlock; blockNum++ {
		block, _ := env.GetBlockAndHeader(blockNum)
		txs := block.Transactions()
		blockAccessListTxs := 0

		for _, tx := range txs {
			if len(tx.GetAccessList()) > 0 {
				blockAccessListTxs++
			}
		}

		if len(txs) > 0 {
			ratio := float64(blockAccessListTxs) / float64(len(txs))
			ratios = append(ratios, ratio)
		}
	}

	return totalTxs, totalAccessListTxs, ratios
}

func TestAccessListSample(t *testing.T) {
	env := helper.PrepareEnv()
	totalBlocks := endNum - startNum
	groupSize := 100
	groupCount := 20

	groups := make([]uint64, groupCount)
	for i := 0; i < groupCount; i++ {
		groups[i] = startNum + uint64(rand.Intn(int(totalBlocks)-groupSize))
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	overallTotalTxs := 0
	overallTotalAccessListTxs := 0
	overallRatios := make([]float64, 0)

	for i, startBlock := range groups {
		wg.Add(1)
		go func(i int, startBlock uint64) {
			defer wg.Done()
			endBlock := startBlock + uint64(groupSize)
			totalTxs, totalAccessListTxs, ratios := sampleAccessListRatio(&env, startBlock, endBlock)

			mu.Lock()
			overallTotalTxs += totalTxs
			overallTotalAccessListTxs += totalAccessListTxs
			overallRatios = append(overallRatios, ratios...)
			mu.Unlock()

			ratio := float64(totalAccessListTxs) / float64(totalTxs)
			fmt.Printf("Group %d: Start Block: %d, End Block: %d, Total Txs: %d, AccessList Txs: %d, Ratio: %.4f\n", i+1, startBlock, endBlock, totalTxs, totalAccessListTxs, ratio)
		}(i, startBlock)
	}

	wg.Wait()

	overallTotalRatio := float64(overallTotalAccessListTxs) / float64(overallTotalTxs)

	sort.Float64s(overallRatios)
	median := overallRatios[len(overallRatios)/2]
	if len(overallRatios)%2 == 0 {
		median = (overallRatios[len(overallRatios)/2-1] + overallRatios[len(overallRatios)/2]) / 2
	}

	variance := 0.0
	for _, ratio := range overallRatios {
		variance += math.Pow(ratio-overallTotalRatio, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(overallRatios)))

	t.Logf("Overall Total Ratio: %f", overallTotalRatio)
	t.Logf("Median: %f", median)
	t.Logf("Standard Deviation: %f", stdDev)
}
