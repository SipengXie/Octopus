package pipeline

import (
	dag "blockConcur/graph"
	"blockConcur/schedule"
	"fmt"
	"sync"
	"time"
)

type MODE int

const (
	BlkConcur MODE = iota
	HESI
	LOBA
	HEFT
	PEFT
	CPTL
	CPOP
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

func Schedule(graph *dag.Graph, useTree bool, numWorker int, mode MODE) (float64, schedule.Processors, uint64, schedule.Method) {
	st := time.Now()
	scheduleAgg := schedule.NewScheduleAggregator(graph, useTree, numWorker)
	var processors schedule.Processors
	var makespan uint64
	var method schedule.Method
	switch mode {
	case BlkConcur:
		// processors, makespan, method = scheduleAgg.Schedule()
		processors, makespan, method = scheduleAgg.ScheduleHEFT()
	case HESI:
		processors, makespan, method = scheduleAgg.ScheduleHESI()
	case LOBA:
		processors, makespan, method = scheduleAgg.ScheduleLOBA()
	case HEFT:
		processors, makespan, method = scheduleAgg.ScheduleHEFT()
	case PEFT:
		processors, makespan, method = scheduleAgg.SchedulePEFT()
	case CPTL:
		processors, makespan, method = scheduleAgg.ScheduleCPTL()
	case CPOP:
		processors, makespan, method = scheduleAgg.ScheduleCPOP()
	default:
		panic("invalid mode")
	}
	cost := time.Since(st).Seconds()
	return cost, processors, makespan, method
}

func (s *Scheduler) Run() {
	var elapsed float64
	for input := range s.InputChan {
		// fmt.Println("Scheduler")
		if input.Flag == END {
			outMessage := &ScheduleMessage{
				Flag: END,
			}
			s.OutputChan <- outMessage
			close(s.OutputChan)
			s.Wg.Done()
			fmt.Println("Parallel Schedule Cost:", elapsed, "s")
			return
		}

		cost, processors, makespan, _ := Schedule(input.Graph, s.UseTree, s.NumWorker, BlkConcur)
		elapsed += cost
		outMessage := &ScheduleMessage{
			Flag:       START,
			Processors: processors,
			Makespan:   makespan,
			PostBlock:  input.PostBlock,
			Header:     input.Header,
			Headers:    input.Headers,
			Withdraws:  input.Withdraws,
		}
		s.OutputChan <- outMessage
	}
}
