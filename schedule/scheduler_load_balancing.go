package schedule

import (
	"container/heap"
	"octopus/graph"
	"octopus/utils"
)

// Without priority and IBP, implemented by queCC
type SchedulerLOBA struct {
	graph      *graph.Graph
	processors Processors
	makespan   uint64
}

func NewSchedulerLOBA(graph *graph.Graph, processors Processors) *SchedulerLOBA {
	return &SchedulerLOBA{
		graph:      graph,
		processors: processors,
		makespan:   0,
	}
}

func (s *SchedulerLOBA) Makespan() uint64 {
	return s.makespan
}

// function Schedule, accoring to the graph, choose task from the graph with the smallest EST as
// the priority, and choose the processor with the smallest eft.
// The processor does not implement the insertion-base policy (IBP), but it is does not need
// as the task is scheduled in the order of EST.
func (s *SchedulerLOBA) Schedule() {
	// using priority queue, the priority is the EST
	tobe_scheduled := make(PriorityTaskQueue, 0)
	mapIndegree := make(map[*utils.ID]uint)
	tmap := make(map[*utils.ID]*TaskWrapper)
	for id, v := range s.graph.Vertices {
		tWrap := &TaskWrapper{
			Task: v.Task,
		}
		tmap[id] = tWrap
		mapIndegree[id] = v.InDegree
		if v.InDegree == 0 {
			// initially, we do not use the priortiy attribute
			tobe_scheduled.Push(&TaskWrapper{
				Task: v.Task,
			})
		}
	}
	heap.Init(&tobe_scheduled)
	for tobe_scheduled.Len() > 0 {
		twarp := heap.Pop(&tobe_scheduled).(*TaskWrapper)
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
			if mapIndegree[succID] == 0 && succID != utils.EndID {
				succTwarp.Priority = succTwarp.EST
				heap.Push(&tobe_scheduled, succTwarp)
			}
		}
	}

	s.makespan = tmap[utils.EndID].EST
}
