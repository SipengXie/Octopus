package test

// 这个测试会测试那些rwset不准确的交易
//当他们被放到区块最后，他们新产生的rwset是否还能使用

import (
	"blockConcur/helper"
	"blockConcur/types"
	"testing"
)

func TestDoubleExecution(t *testing.T) {
	env := helper.PrepareEnv()
	dbTx, err := env.DB.BeginRo(env.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer dbTx.Rollback()

	for blockNum := startNum; blockNum < endNum; blockNum++ {
		block, header := env.GetBlockAndHeader(uint64(blockNum))
		ibs1 := env.GetIBS(uint64(blockNum), dbTx)
		ibs2 := env.GetIBS(uint64(blockNum), dbTx)
		headers := env.FetchHeaders(blockNum-256, blockNum)

		// 生成准确的rwset
		accurateTasks := helper.GenerateAccurateRwSets(block.Transactions(), header, headers, ibs1, convertNum)

		// 生成预测的rwset
		predictTasks := helper.GeneratePredictRwSets(block.Transactions(), header, headers, ibs2, convertNum)

		// 找出rwset不准确的交易
		inaccurateTxs := findInaccurateTxs(accurateTasks, predictTasks)

		// 先串行执行txs，遇到inaccurateTxs，我们不提交，收集他们的新RwSet_1；遇到正常tx，我们提交
		// 然后，我们将所有的inAccurateTxs放到最后执行，并收集新新RwSet_2，比较RwSet_1和RwSet_2

	}
}

func findInaccurateTxs(accurateTasks, predictTasks []*types.Task) []int {
	inaccurateTxs := []int{}
	for i := range accurateTasks {
		if !compareRwSets(accurateTasks[i].RwSet, predictTasks[i].RwSet) {
			inaccurateTxs = append(inaccurateTxs, i)
		}
	}
	return inaccurateTxs
}
