#!/bin/bash

for procNum in 2 4 8 16 32
do
    export PROCESSOR_NUM=$procNum
    
    echo "运行测试 processorNum: $procNum"

    go test -run ^TestSingleBlock$ blockConcur/test -v -timeout 30m -count=1 2>&1 | tee "./res/single_block_proc${procNum}.txt"

done
