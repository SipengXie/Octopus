package test

import (
	"blockConcur/helper"
	"blockConcur/rwset"
	"fmt"
	"testing"

	"golang.org/x/exp/rand"
)

func TestRwSetAccuracy(t *testing.T) {
	env := helper.PrepareEnv()

	groupSize := 100
	groupCount := 20
	totalBlocks := int(endNum - startNum)

	groups := make([]uint64, groupCount)
	for i := 0; i < groupCount; i++ {
		for {
			startBlock := startNum + uint64(rand.Intn(totalBlocks-groupSize))
			overlap := false
			for j := 0; j < i; j++ {
				if startBlock >= groups[j] && startBlock < groups[j]+uint64(groupSize) {
					overlap = true
					break
				}
			}
			if !overlap {
				groups[i] = startBlock
				break
			}
		}
	}
	results := make(chan string, groupCount)
	for i := 0; i < groupCount; i++ {
		go func(group int) {
			dbTx, err := env.DB.BeginRo(env.Ctx)
			if err != nil {
				results <- fmt.Sprintf("Group %d failed to create dbtx: %v", group+1, err)
				return
			}
			defer dbTx.Rollback()

			startBlock := groups[group]
			endBlock := startBlock + uint64(groupSize)

			totalTxs := 0
			mismatchTxs := 0

			for blockNum := startBlock; blockNum < endBlock; blockNum++ {
				ibs1 := env.GetIBS(blockNum, dbTx)
				ibs2 := env.GetIBS(blockNum, dbTx)
				block, header := env.GetBlockAndHeader(blockNum)
				txs := block.Transactions()
				headers := env.FetchHeaders(blockNum-256, blockNum)

				totalTxs += len(txs)

				accurateTasks := helper.GenerateAccurateRwSets(txs, header, headers, ibs1, convertNum)
				predictTasks := helper.GeneratePredictRwSets(txs, header, headers, ibs2, convertNum)

				for i, accurateTask := range accurateTasks {
					predictTask := predictTasks[i]
					if !compareRwSets(accurateTask.RwSet, predictTask.RwSet) {
						mismatchTxs++
					}
				}
			}

			mismatchRatio := float64(mismatchTxs) / float64(totalTxs)
			results <- fmt.Sprintf("Group %d (Blocks %d - %d) Mismatch ratio between predicted and accurate RwSets: %.2f%%", group+1, startBlock, endBlock-1, mismatchRatio*100)
		}(i)
	}

	for i := 0; i < groupCount; i++ {
		t.Log(<-results)
	}
}

func compareRwSets(accurateRwSet, predictRwSet *rwset.RwSet) bool {
	return compareSet(accurateRwSet.ReadSet, predictRwSet.ReadSet) &&
		compareSet(accurateRwSet.WriteSet, predictRwSet.WriteSet)
}

func compareSet(accurateSet, predictSet map[string]struct{}) bool {
	if len(accurateSet) != len(predictSet) {
		return false
	}

	for key := range accurateSet {
		if _, exists := predictSet[key]; !exists {
			return false
		}
	}

	return true
}
