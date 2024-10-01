package schedule

import (
	"blockConcur/graph"
	"blockConcur/types"
	"blockConcur/utils" // Assuming this import is necessary for utils.NewID
	"fmt"
	"sync"
	"testing"

	"github.com/ledgerwatch/erigon-lib/common"
)

//TODO: Add test for the schedule algorithm
// Schedule a graph using schedule_aggregator.go (test both tree and list)
// Schedule a graph using schudule_heuristic_simple.go
// Schedule a graph using schedule_load_balancing.go

const thread_num = 3

func generateTestGraph() *graph.Graph {
	graph := graph.NewGraph()
	taskList := make([]*types.Task, 10)

	id0 := utils.NewID(0, 0, 0)
	id1 := utils.NewID(1, 0, 0)
	id2 := utils.NewID(2, 0, 0)
	id3 := utils.NewID(3, 0, 0)
	id4 := utils.NewID(4, 0, 0)
	id5 := utils.NewID(5, 0, 0)
	id6 := utils.NewID(6, 0, 0)
	id7 := utils.NewID(7, 0, 0)
	id8 := utils.NewID(8, 0, 0)
	id9 := utils.NewID(9, 0, 0)

	taskList[0] = types.NewTask(id0, 13, nil, common.Hash{}, common.Hash{})
	taskList[1] = types.NewTask(id1, 17, nil, common.Hash{}, common.Hash{})
	taskList[2] = types.NewTask(id2, 14, nil, common.Hash{}, common.Hash{})
	taskList[3] = types.NewTask(id3, 13, nil, common.Hash{}, common.Hash{})
	taskList[4] = types.NewTask(id4, 12, nil, common.Hash{}, common.Hash{})
	taskList[5] = types.NewTask(id5, 13, nil, common.Hash{}, common.Hash{})
	taskList[6] = types.NewTask(id6, 11, nil, common.Hash{}, common.Hash{})
	taskList[7] = types.NewTask(id7, 10, nil, common.Hash{}, common.Hash{})
	taskList[8] = types.NewTask(id8, 17, nil, common.Hash{}, common.Hash{})
	taskList[9] = types.NewTask(id9, 15, nil, common.Hash{}, common.Hash{})

	for _, task := range taskList {
		graph.AddVertex(task)
	}

	graph.AddEdge(id0, id1)
	graph.AddEdge(id0, id2)
	graph.AddEdge(id0, id3)
	graph.AddEdge(id0, id4)
	graph.AddEdge(id0, id5)

	graph.AddEdge(id1, id7)
	graph.AddEdge(id1, id8)

	graph.AddEdge(id2, id6)

	graph.AddEdge(id3, id7)
	graph.AddEdge(id3, id8)

	graph.AddEdge(id4, id8)

	graph.AddEdge(id5, id7)

	graph.AddEdge(id6, id9)

	graph.AddEdge(id7, id9)

	graph.AddEdge(id8, id9)

	graph.GenerateVirtualVertex()
	graph.GenerateProperties()
	return graph
}

func TestListEFT(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}

	eftScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	eftScheduler.listSchedule(EFT, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(eftScheduler.makespan)
}

func TestListCPOP(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}

	cpopScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	cpopScheduler.pqSchedule(CPOP, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(cpopScheduler.makespan)
}

func TestListCPTL(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}

	cptlScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	cptlScheduler.pqSchedule(CPTL, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(cptlScheduler.makespan)
}

func TestListCT(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}

	ctScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	ctScheduler.listSchedule(CT, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(ctScheduler.makespan)
}

func TestListHESI(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}

	hesiScheduler := NewSchedulerHESI(graph, processors)
	hesiScheduler.schedule()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(hesiScheduler.makespan)
}

func TestListLOBA(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorList()
	}
	lobaScheduler := NewSchedulerLOBA(graph, processors)
	lobaScheduler.schedule()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(lobaScheduler.makespan)
}

func TestTreeEFT(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorTree()
	}

	eftScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	eftScheduler.listSchedule(EFT, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(eftScheduler.makespan)
}

func TestTreeCT(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorTree()
	}

	ctScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	ctScheduler.listSchedule(CT, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(ctScheduler.makespan)
}

func TestTreeCPTL(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorTree()
	}

	cptlScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	cptlScheduler.pqSchedule(CPTL, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(cptlScheduler.makespan)
}

func TestTreeCPOP(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorTree()
	}

	cpopScheduler := NewSchedulerHeur(graph, processors)
	var wg sync.WaitGroup
	wg.Add(1)
	cpopScheduler.pqSchedule(CPOP, &wg)
	wg.Wait()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(cpopScheduler.makespan)
}

func TestSimpleHESI(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorSimple()
	}

	hesiScheduler := NewSchedulerHESI(graph, processors)
	hesiScheduler.schedule()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(hesiScheduler.makespan)
}

func TestSimpleLOBA(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	processors := make(Processors, thread_num)
	for i := 0; i < thread_num; i++ {
		processors[i] = NewProcessorSimple()
	}
	lobaScheduler := NewSchedulerLOBA(graph, processors)
	lobaScheduler.schedule()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(lobaScheduler.makespan)
}

// Test the schedule aggregator
func TestScheduleAggregator(t *testing.T) {
	t.Parallel()
	graph := generateTestGraph()
	sa := NewScheduleAggregator(graph, true, thread_num)
	processors, makespan, method := sa.Schedule()
	for _, processor := range processors {
		processor.Print()
	}
	fmt.Println(makespan, method)
}
