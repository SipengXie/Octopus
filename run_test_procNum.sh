#!/bin/bash

for procNum in 2 4 8 16 32
do
    export PROCESSOR_NUM=$procNum
    
    echo "运行测试 processorNum: $procNum"

    go test -run ^TestSingleBlock$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/blkConcur${procNum}.txt"
    go test -run ^TestSingleBlockPredict$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/blkPredict${procNum}.txt"
    go test -run ^TestSingleBlockOCCDA$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/OCCDA${procNum}.txt"
    go test -run ^TestSingleBlockQuecc$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/QUECC${procNum}.txt"

done
