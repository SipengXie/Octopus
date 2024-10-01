package schedule

import (
	"blockConcur/graph"
	"sync"
)

type ScheduleAggregator struct {
	graph     *graph.Graph
	use_tree  bool
	numWorker int
}

func NewScheduleAggregator(graph *graph.Graph, use_tree bool, numWorker int) *ScheduleAggregator {
	return &ScheduleAggregator{
		graph:     graph,
		use_tree:  use_tree,
		numWorker: numWorker,
	}
}

func (sa *ScheduleAggregator) Schedule() (Processors, uint64, Method) {
	// we have 4 algorithm to choose from
	processors := make([]Processors, 4)
	for i := 0; i < 4; i++ {
		processors[i] = make(Processors, sa.numWorker)
		for j := 0; j < sa.numWorker; j++ {
			if sa.use_tree {
				processors[i][j] = NewProcessorTree()
			} else {
				processors[i][j] = NewProcessorList()
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(4)
	schedulers := make([]*SchedulerHeur, 4)

	go func() {
		schedulers[0] = NewSchedulerHeur(sa.graph, processors[0])
		schedulers[0].listSchedule(EFT, &wg)
	}()

	go func() {
		schedulers[1] = NewSchedulerHeur(sa.graph, processors[1])
		schedulers[1].listSchedule(CT, &wg)
	}()

	go func() {
		schedulers[2] = NewSchedulerHeur(sa.graph, processors[2])
		schedulers[2].pqSchedule(CPTL, &wg)
	}()

	go func() {
		schedulers[3] = NewSchedulerHeur(sa.graph, processors[3])
		schedulers[3].pqSchedule(CPOP, &wg)
	}()

	wg.Wait()

	minMakespan := schedulers[0].makespan
	retProcessors := processors[0]
	method := EFT
	for i := 1; i < 4; i++ {
		if schedulers[i].makespan < minMakespan {
			minMakespan = schedulers[i].makespan
			retProcessors = processors[i]
			method = Method(i)
		}
	}
	return retProcessors, minMakespan, method
}
