package test

import (
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
)

const startNum uint64 = 18511000
const endNum uint64 = 20000000
const cacheSize = 8192 << 1
const use_tree_threshold = 10000
const early_abort bool = true

// for signle block, fetchPool, ivPool and processors will not content on the
// cpu resources.
var fetchPoolSize = runtime.NumCPU() / 2
var ivPoolSize = runtime.NumCPU() / 2
var processorNum = 2
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
