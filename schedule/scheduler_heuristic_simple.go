package schedule

import (
	"container/heap"
	"octopus/graph"
	"octopus/utils"
)

// Without IBP, using simple priority, implemented by octopus
type SchedulerHESI struct {
	graph      *graph.Graph
	processors Processors
	makespan   uint64
}

func NewSchedulerHESI(graph *graph.Graph, processors Processors) *SchedulerHESI {
	return &SchedulerHESI{
		graph:      graph,
		processors: processors,
		makespan:   0,
	}
}

func (s *SchedulerHESI) Makespan() uint64 {
	return s.makespan
}

func (s *SchedulerHESI) Schedule() {
	pq := make(PriorityTaskQueue, 0)
	// prioritze tasks with it's cost
	tmap := make(map[*utils.ID]*TaskWrapper)
	mapIndegree := make(map[*utils.ID]uint)
	for id, v := range s.graph.Vertices {
		tWrap := &TaskWrapper{
			Task:     v.Task,
			Priority: ^v.Task.Cost,
		}
		tmap[id] = tWrap
		mapIndegree[id] = v.InDegree
		if v.InDegree == 0 {
			heap.Push(&pq, tmap[id])
		}
	}

	for pq.Len() > 0 {
		twarp := heap.Pop(&pq).(*TaskWrapper)
		if twarp.Task.Tid != utils.EndID && twarp.Task.Tid != utils.SnapshotID {
			var tempValue eftResult
			var processor Processor
			for _, p := range s.processors {
				result := p.FindEFT(twarp)
				if result.IsLessThan(tempValue) {
					tempValue = result
					processor = p
				}
			}
			twarp.EFT = tempValue.EFT()
			processor.AddTask(twarp, tempValue)
		}

		for succID := range s.graph.AdjacencyMap[twarp.Task.Tid] {
			mapIndegree[succID]--
			// update EST
			succTwarp := tmap[succID]
			succTwarp.EST = max(succTwarp.EST, twarp.EFT)
			if mapIndegree[succID] == 0 {
				heap.Push(&pq, tmap[succID])
			}
		}
	}

	s.makespan = tmap[utils.EndID].EST
}
