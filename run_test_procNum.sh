#!/bin/bash

for procNum in 2 4 8 16 32
do
    export PROCESSOR_NUM=$procNum
    
    echo "运行测试 processorNum: $procNum"
    go test -run ^TestRealSchedule$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/realSchedule${procNum}.txt"
    # go test -run ^TestSingleBlock$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/blkConcur${procNum}.txt"
    # go test -run ^TestSingleBlockPredict$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/blkPredict${procNum}.txt"
    # go test -run ^TestSingleBlockOCCDA$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/OCCDA${procNum}.txt"
    # go test -run ^TestSingleBlockQUECC$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/QUECC${procNum}.txt"
    # go test -run ^TestPipeline$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/pipeline${procNum}.txt"
done
# go test -run ^TestTreeList$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/treeList.txt"
