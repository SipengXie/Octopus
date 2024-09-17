package graph

import (
	mv "blockConcur/multiversion"
	"blockConcur/types"
)

type NodeId *mv.GlobalId

type Vertex struct {
	Task      *types.Task
	InDegree  uint // IN-DEGREE
	OutDegree uint // OUT-DEGREE

	// properties needed to schedule
	Rank_u uint64
	Rank_d uint64
	CT     uint64
}

// UndirectedGraph
type Graph struct {
	Vertices     map[NodeId]*Vertex             `json:"vertices"`
	AdjacencyMap map[NodeId]map[NodeId]struct{} `json:"adjacencyMap"`
	ReverseMap   map[NodeId]map[NodeId]struct{} `json:"reverseMap"`

	CriticalPathLen uint64
}

func NewGraph() *Graph {
	return &Graph{
		Vertices:     make(map[NodeId]*Vertex),
		AdjacencyMap: make(map[NodeId]map[NodeId]struct{}),
		ReverseMap:   make(map[NodeId]map[NodeId]struct{}),
	}
}

func (g *Graph) AddVertex(task *types.Task) {
	id := task.Tid
	_, exist := g.Vertices[id]
	if exist {
		return
	}
	v := &Vertex{
		Task: task,
	}
	g.Vertices[id] = v
	g.AdjacencyMap[id] = make(map[NodeId]struct{})
	g.ReverseMap[id] = make(map[NodeId]struct{})
}

func (g *Graph) AddEdge(source, destination NodeId) {
	if g.HasEdge(source, destination) {
		return
	}
	g.AdjacencyMap[source][destination] = struct{}{}
	g.ReverseMap[destination][source] = struct{}{}
	g.Vertices[source].OutDegree++
	g.Vertices[destination].InDegree++
}

func (g *Graph) HasEdge(source, destination NodeId) bool {
	_, ok := g.Vertices[source]
	if !ok {
		return false
	}

	_, ok = g.Vertices[destination]
	if !ok {
		return false
	}

	_, ok = g.AdjacencyMap[source][destination]
	return ok
}

func (g *Graph) getTopo(rev bool) []NodeId {
	mapDegree := make(map[NodeId]uint)
	degreeZero := make([]NodeId, 0)
	for id, v := range g.Vertices {
		if rev {
			mapDegree[id] = v.OutDegree
		} else {
			mapDegree[id] = v.InDegree
		}
		if mapDegree[id] == 0 {
			degreeZero = append(degreeZero, id)
		}
	}

	topo := make([]NodeId, 0)
	for {
		newDegreeZero := make([]NodeId, 0)
		for _, vid := range degreeZero {
			topo = append(topo, vid)
			edges := g.AdjacencyMap[vid]
			if rev {
				edges = g.ReverseMap[vid]
			}

			for succId := range edges {
				mapDegree[succId]--
				if mapDegree[succId] == 0 {
					newDegreeZero = append(newDegreeZero, succId)
				}
			}
		}
		degreeZero = newDegreeZero
		if len(degreeZero) == 0 {
			break
		}
	}
	return topo
}

func (g *Graph) calcRankD() {
	topo := g.getTopo(false)
	stid := topo[0]
	g.Vertices[stid].Rank_d = 0
	for i := 1; i < len(topo); i++ {
		vid := topo[i]
		curv := g.Vertices[vid]
		// getmaxPredcessor
		maxPred := uint64(0)
		for predid := range g.ReverseMap[vid] {
			pred := g.Vertices[predid]
			maxPred = max(maxPred, pred.Rank_d+pred.Task.Cost)
		}
		curv.Rank_d = maxPred
	}
}

func (g *Graph) calcRankUCT() {
	topo := g.getTopo(true)
	edid := topo[0]
	g.Vertices[edid].CT = g.Vertices[edid].Task.Cost
	g.Vertices[edid].Rank_u = g.Vertices[edid].Task.Cost
	for i := 0; i < len(topo); i++ {
		vid := topo[i]
		cur := g.Vertices[vid]
		// getmaxSuccessor
		maxRanku := uint64(0)
		maxct := uint64(0)

		for succid := range g.AdjacencyMap[vid] {
			succ := g.Vertices[succid]
			maxRanku = max(maxRanku, succ.Rank_u)
			maxct = max(maxct, succ.CT+succ.Task.Cost)
		}
		cur.Rank_u = maxRanku + cur.Task.Cost
		cur.CT = maxct
	}
}

func (g *Graph) GenerateProperties() {
	g.CriticalPathLen = 0

	g.calcRankD()
	g.calcRankUCT()

	for _, v := range g.Vertices {
		g.CriticalPathLen = max(g.CriticalPathLen, v.Rank_u+v.Rank_d)
	}
}
