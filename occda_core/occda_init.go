package occda_core

import (
	"container/heap"

	"octopus/graph"
	"octopus/utils"
)

// HeapSid definition
type HeapSid []*OCCDATask

func (h HeapSid) Len() int           { return len(h) }
func (h HeapSid) Less(i, j int) bool { return h[i].sid.Compare(h[j].sid) < 0 }
func (h HeapSid) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *HeapSid) Push(x interface{}) {
	*h = append(*h, x.(*OCCDATask))
}
func (h *HeapSid) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Initialize the heap
func OCCDAInitialize(occdaTasks []*OCCDATask, g *graph.Graph) (*HeapSid, map[*utils.ID]int) {
	h := &HeapSid{}
	// generate a map that maps tid to task_idx
	tidToTaskIdx := make(map[*utils.ID]int)
	for i, task := range occdaTasks {
		tidToTaskIdx[task.Tid] = i
	}
	tidToTaskIdx[utils.SnapshotID] = -1
	// push tasks into heap
	if g == nil {
		for _, occdaTask := range occdaTasks {
			occdaTask.sid = utils.SnapshotID
			heap.Push(h, occdaTask)
		}
	} else {
		for _, occdaTask := range occdaTasks {
			sid_max := utils.SnapshotID
			// find the max dependency by checking the reverseMap
			edges := g.ReverseMap[occdaTask.Tid]
			for to := range edges {
				if to.Compare(sid_max) > 0 {
					sid_max = to
				}
			}

			occdaTask.sid = sid_max
			heap.Push(h, occdaTask)
		}
	}
	return h, tidToTaskIdx
}
