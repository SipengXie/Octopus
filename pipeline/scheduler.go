package pipeline

import (
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

		scheduleAgg := schedule.NewScheduleAggregator(input.Graph, s.UseTree, s.NumWorker)
		st := time.Now()
		processors, makespan, _ := scheduleAgg.Schedule()
		// If in debug mode, we should output the method to analyze which one performs best
		elapsed += time.Since(st).Milliseconds()
		outMessage := &ScheduleMessage{
			Flag:       START,
			Processors: processors,
			Makespan:   makespan,
		}
		s.OutputChan <- outMessage
	}
}
