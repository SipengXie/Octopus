package test

import "runtime"

const startNum uint64 = 18500000
const endNum uint64 = 18500010
const cacheSize = 8192
const use_tree_threshold = 10000
const early_abort bool = false

// for signle block, fetchPool, ivPool and processors will not content on the
// cpu resources.
var fetchPoolSize = runtime.NumCPU() / 2
var ivPoolSize = runtime.NumCPU() / 2
var processorNum = runtime.NumCPU()
var convertNum = runtime.NumCPU()
var use_tree = func(i int) bool {
	return i >= use_tree_threshold
}
