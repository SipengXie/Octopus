package schedule

import (
	"blockConcur/graph"
	"blockConcur/utils"
	"container/heap"
	"sync"
)

const MAXUINT64 = ^uint64(0) >> 1

type Method int

func (m Method) String() string {
	switch m {
	case EFT:
		return "EFT"
	case CT:
		return "CT"
	case CPTL:
		return "CPTL"
	case CPOP:
		return "CPOP"
	case HESI:
		return "HESI"
	case LOBA:
		return "LOBA"
	default:
		return "Unknown"
	}
}

const (
	EFT Method = iota
	CT
	CPTL
	CPOP
	HESI
	LOBA
)

type SchedulerHeur struct {
	graph      *graph.Graph
	processors Processors
	makespan   uint64
}

func NewSchedulerHeur(graph *graph.Graph, processors Processors) *SchedulerHeur {
	return &SchedulerHeur{
		graph:      graph,
		processors: processors,
		makespan:   0,
	}
}

type tpResult struct {
	timespan uint64
	pq       PriorityTaskQueue
	tMap     map[*utils.ID]*TaskWrapper
}

func (s *SchedulerHeur) taskPrioritize(m Method) *tpResult {
	tMap := make(map[*utils.ID]*TaskWrapper)
	pq := make(PriorityTaskQueue, 0)
	var timespan uint64 = 0
	for _, v := range s.graph.Vertices {
		priority := uint64(0)
		switch m {
		case EFT:
			priority = v.Rank_u
		case CT:
			priority = v.CT
		}
		tWrap := &TaskWrapper{
			Task:     v.Task,
			Priority: priority,
			AST:      0,
			EST:      0,
			EFT:      0,
		}
		heap.Push(&pq, tWrap)
		tMap[v.Task.Tid] = tWrap
		timespan += v.Task.Cost
	}
	return &tpResult{
		timespan: timespan,
		pq:       pq,
		tMap:     tMap,
	}
}

func (s *SchedulerHeur) processorAllocation(tpInput *tpResult) {
	for _, p := range s.processors {
		p.SetTimespan(tpInput.timespan)
	}
	for tpInput.pq.Len() > 0 {
		tWrap := heap.Pop(&tpInput.pq).(*TaskWrapper)
		s.selectBestProcessor(tWrap)

		for succID := range s.graph.AdjacencyMap[tWrap.Task.Tid] {
			succTwrap := tpInput.tMap[succID]
			succTwrap.EST = max(succTwrap.EST, tWrap.EFT)
		}

	}
	s.makespan = tpInput.tMap[utils.EndID].EST
}

func (s *SchedulerHeur) selectBestProcessor(tWrap *TaskWrapper) {
	if tWrap.Task.Tid == utils.SnapshotID || tWrap.Task.Tid == utils.EndID {
		return
	}
	var pid int = 0
	var tempValue eftResult // TODO: implement
	for id, p := range s.processors {
		res := p.FindEFT(tWrap)
		if res.IsLessThan(tempValue) {
			pid = id
			tempValue = res
		}
	}
	tWrap.EFT = tempValue.EFT()
	tWrap.AST = tWrap.EFT - tWrap.Task.Cost
	s.processors[pid].AddTask(tWrap, tempValue)
}

func (s *SchedulerHeur) listSchedule(m Method, wg *sync.WaitGroup) {
	defer wg.Done()
	s.processorAllocation(s.taskPrioritize(m))

}

func (s *SchedulerHeur) pqSchedule(m Method, wg *sync.WaitGroup) {
	defer wg.Done()
	tMap := make(map[*utils.ID]*TaskWrapper)
	isCP := make(map[*utils.ID]struct{})
	mapIndegree := make(map[*utils.ID]uint)
	var timespan uint64 = 0
	for _, v := range s.graph.Vertices {
		var priority uint64
		switch m {
		case CPTL:
			priority = s.graph.CriticalPathLen - v.Rank_d
		case CPOP:
			priority = v.Rank_d + v.Rank_u
		}

		if priority == s.graph.CriticalPathLen {
			isCP[v.Task.Tid] = struct{}{}
		}
		timespan += v.Task.Cost
		mapIndegree[v.Task.Tid] = v.InDegree

		tWrap := &TaskWrapper{
			Task:     v.Task,
			Priority: priority,
			AST:      0,
			EST:      0,
			EFT:      0,
		}
		tMap[v.Task.Tid] = tWrap
	}
	for _, p := range s.processors {
		p.SetTimespan(timespan)
	}

	cpProcesser := s.processors[0]
	tEntry := tMap[utils.SnapshotID]
	pq := make(PriorityTaskQueue, 0)
	heap.Push(&pq, tEntry)

	for pq.Len() != 0 {
		tWrap := heap.Pop(&pq).(*TaskWrapper)
		if _, ok := isCP[tWrap.Task.Tid]; ok && tWrap.Task.Tid != utils.SnapshotID && tWrap.Task.Tid != utils.EndID {
			res := cpProcesser.FindEFT(tWrap)
			tWrap.EFT = res.EFT()
			tWrap.AST = tWrap.EFT - tWrap.Task.Cost
			cpProcesser.AddTask(tWrap, res)
		} else {
			s.selectBestProcessor(tWrap)
		}

		for succID := range s.graph.AdjacencyMap[tWrap.Task.Tid] {
			succTwrap := tMap[(succID)]
			succTwrap.EST = max(succTwrap.EST, tWrap.EFT)
			mapIndegree[(succID)]--
			if mapIndegree[(succID)] == 0 && succID != utils.EndID {
				heap.Push(&pq, succTwrap)
			}
		}
	}
	s.makespan = tMap[utils.EndID].EST

}
