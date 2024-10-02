package pipeline

import (
	dag "blockConcur/graph"
	"blockConcur/schedule"
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	NumWorker  int
	UseTree    bool
	Wg         *sync.WaitGroup
	InputChan  chan *GraphMessage
	OutputChan chan *ScheduleMessage
}

func NewScheduler(numWorker int, useTree bool, wg *sync.WaitGroup, in chan *GraphMessage, out chan *ScheduleMessage) *Scheduler {
	return &Scheduler{
		NumWorker:  numWorker,
		UseTree:    useTree,
		Wg:         wg,
		InputChan:  in,
		OutputChan: out,
	}
}

func Schedule(graph *dag.Graph, useTree bool, numWorker int) (int64, schedule.Processors, uint64, schedule.Method) {
	st := time.Now()
	scheduleAgg := schedule.NewScheduleAggregator(graph, useTree, numWorker)
	processors, makespan, method := scheduleAgg.Schedule()
	cost := time.Since(st).Milliseconds()
	return cost, processors, makespan, method
}

func (s *Scheduler) Run() {
	var elapsed int64
	for input := range s.InputChan {
		// fmt.Println("Scheduler")
		if input.Flag == END {
			outMessage := &ScheduleMessage{
				Flag: END,
			}
			s.OutputChan <- outMessage
			close(s.OutputChan)
			s.Wg.Done()
			fmt.Println("Parallel Schedule Cost:", elapsed, "ms")
			return
		}

		cost, processors, makespan, _ := Schedule(input.Graph, s.UseTree, s.NumWorker)
		elapsed += cost
		outMessage := &ScheduleMessage{
			Flag:       START,
			Processors: processors,
			Makespan:   makespan,
		}
		s.OutputChan <- outMessage
	}
}
