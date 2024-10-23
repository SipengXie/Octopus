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

// not offical product, only for benchmark
func (sa *ScheduleAggregator) ScheduleHESI() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	for j := 0; j < sa.numWorker; j++ {
		processors[j] = NewProcessorSimple()
	}

	scheduler := NewSchedulerHESI(sa.graph, processors)
	scheduler.Schedule()

	return processors, scheduler.makespan, HESI
}

// not offical product, only for benchmark
func (sa *ScheduleAggregator) ScheduleLOBA() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	for j := 0; j < sa.numWorker; j++ {
		processors[j] = NewProcessorSimple()
	}

	scheduler := NewSchedulerLOBA(sa.graph, processors)
	scheduler.Schedule()

	return processors, scheduler.makespan, LOBA
}

func (sa *ScheduleAggregator) ScheduleHEFT() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	if sa.use_tree {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorTree()
		}
	} else {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorList()
		}
	}

	scheduler := NewSchedulerHeur(sa.graph, processors)
	wg := sync.WaitGroup{}
	wg.Add(1)
	scheduler.listSchedule(HEFT, &wg)

	return processors, scheduler.makespan, HEFT
}

func (sa *ScheduleAggregator) SchedulePEFT() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	if sa.use_tree {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorTree()
		}
	} else {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorList()
		}
	}

	scheduler := NewSchedulerHeur(sa.graph, processors)
	wg := sync.WaitGroup{}
	wg.Add(1)
	scheduler.listSchedule(PEFT, &wg)

	return processors, scheduler.makespan, PEFT
}

func (sa *ScheduleAggregator) ScheduleCPTL() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	if sa.use_tree {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorTree()
		}
	} else {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorList()
		}
	}

	scheduler := NewSchedulerHeur(sa.graph, processors)
	wg := sync.WaitGroup{}
	wg.Add(1)
	scheduler.pqSchedule(CPTL, &wg)

	return processors, scheduler.makespan, CPTL
}

func (sa *ScheduleAggregator) ScheduleCPOP() (Processors, uint64, Method) {
	processors := make(Processors, sa.numWorker)
	if sa.use_tree {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorTree()
		}
	} else {
		for j := 0; j < sa.numWorker; j++ {
			processors[j] = NewProcessorList()
		}
	}

	scheduler := NewSchedulerHeur(sa.graph, processors)
	wg := sync.WaitGroup{}
	wg.Add(1)
	scheduler.pqSchedule(CPOP, &wg)

	return processors, scheduler.makespan, CPOP
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
		schedulers[0].listSchedule(HEFT, &wg)
	}()

	go func() {
		schedulers[1] = NewSchedulerHeur(sa.graph, processors[1])
		schedulers[1].listSchedule(PEFT, &wg)
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
	method := HEFT
	for i := 1; i < 4; i++ {
		if schedulers[i].makespan < minMakespan {
			minMakespan = schedulers[i].makespan
			retProcessors = processors[i]
			method = Method(i)
		}
	}
	return retProcessors, minMakespan, method
}
