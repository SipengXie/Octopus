package test

import (
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
)

const startNum uint64 = 18994147
const endNum uint64 = 18994149
const cacheSize = 8192
const use_tree_threshold = 10000

// TODO: 目前我们必须要Early_abort,因为如果一个意料之外的coinbase地址出现
// 我们会fetch_prize，但这个prize_chain之前的版本可能还在这个调度里pending，从而死锁
// 如果我们把getprize的逻辑改成从cold_state里面的prize_version里面取，我们就还有改进空间
// prize_chain就不需要的换head了，但cold_state里的prize_version是一个读写共用的version
// 当tx commit的时候，prize_version里的所有data都要变成0，这样就不会重复计算peize
// 在predict里，如果出现意想不到的getprize，我们可以就简单的从head到tid遍历一下，反正这个交易提交不了，随便运行即可。
// 当然，在commit for serial里，需要把head到tid-1的data都置为0
const early_abort bool = false

// for signle block, fetchPool, ivPool and processors will not content on the
// cpu resources.
var fetchPoolSize = runtime.NumCPU()
var ivPoolSize = runtime.NumCPU()
var processorNum = 4
var convertNum = runtime.NumCPU()
var use_tree = func(i int) bool {
	return i >= use_tree_threshold
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

func GetProcessorNumFromEnv() int {
	processorNumStr := os.Getenv("PROCESSOR_NUM")
	if processorNumStr != "" {
		if num, err := strconv.Atoi(processorNumStr); err == nil {
			return num
		}
	}
	return processorNum
}

// Get START_NUM from environment variable
func GetStartNumFromEnv() uint64 {
	startNumStr := os.Getenv("START_NUM")
	if startNumStr != "" {
		if num, err := strconv.ParseUint(startNumStr, 10, 64); err == nil {
			return num
		}
	}
	return startNum
}

// Get END_NUM from environment variable
func GetEndNumFromEnv() uint64 {
	endNumStr := os.Getenv("END_NUM")
	if endNumStr != "" {
		if num, err := strconv.ParseUint(endNumStr, 10, 64); err == nil {
			return num
		}
	}
	return endNum
}
