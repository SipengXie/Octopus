package pipeline

import (
	dag "blockConcur/graph"
	"blockConcur/rwset"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"sync"
	"time"
)

type GraphBuilder struct {
	Wg         *sync.WaitGroup
	InputChan  chan *BuildGraphMessage
	OutputChan chan *GraphMessage
}

func NewGraphBuilder(wg *sync.WaitGroup, in chan *BuildGraphMessage, out chan *GraphMessage) *GraphBuilder {
	return &GraphBuilder{
		Wg:         wg,
		InputChan:  in,
		OutputChan: out,
	}
}

func GenerateGraph(tasks types.Tasks, rwAccessedBy *rwset.RwAccessedBy) (float64, *dag.Graph) {
	st := time.Now()
	graph := dag.NewGraph()
	readBy := rwAccessedBy.ReadBy
	writeBy := rwAccessedBy.WriteBy

	for _, task := range tasks {
		graph.AddVertex(task)
	}

	for key := range readBy {
		// get sorted txIds
		rTasks := readBy.TxIds(key)
		wTasks := writeBy.TxIds(key)
		if len(wTasks) == 0 {
			continue
		}
		_, hash := utils.ParseKey(key)
		if hash == utils.PRIZE {
			// The reason for adding all edges is that we consider concurrency optimization for PRIZE.
			// PRIZE read is dependent for all previous write tasks.
			for _, rID := range rTasks {
				for _, wID := range wTasks {
					if rID.Less(wID) || rID.Equal(wID) {
						break
					}
					// we do not read the prize from a single version, so we do not need to record the version
					graph.AddEdge(wID, rID)
				}
			}
		} else {
			// we only add dependency for the closest write task.
			// because the task will only read the latest data.
			for _, rID := range rTasks {
				idx, ok := wTasks.Find(rID)
				if ok {
					idx--
				}
				if idx < 0 {
					continue
				}
				pvwID := wTasks[idx]
				// if ok, it means wTasks[idx] = rTaskID, so we need the previous write task.
				// However, the idx should not be 0.
				// add edge from the previous write task to the read task, and change the task's read version to the previous write version
				graph.AddEdge(pvwID, rID)
				rNode := graph.Vertices[rID]
				pvwNode := graph.Vertices[pvwID]
				rNode.Task.AddReadVersion(key, pvwNode.Task.WriteVersions[key])
			}
		}
	}
	graph.GenerateVirtualVertex()
	graph.GenerateProperties()
	cost := time.Since(st).Seconds()
	return cost, graph
}

func (g *GraphBuilder) Run() {
	var elapsed float64
	for input := range g.InputChan {
		if input.Flag == END {
			outMessage := &GraphMessage{
				Flag: END,
			}
			g.OutputChan <- outMessage
			close(g.OutputChan)
			g.Wg.Done()
			fmt.Println("Graph Generation Cost:", elapsed, "ms")
			return
		}

		cost, graph := GenerateGraph(input.Tasks, input.RwAccessedBy)
		elapsed += cost

		outMessage := &GraphMessage{
			Flag:  START,
			Graph: graph,
		}
		g.OutputChan <- outMessage
	}
}
