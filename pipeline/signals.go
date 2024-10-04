package pipeline

import (
	dag "blockConcur/graph"
	"blockConcur/rwset"
	"blockConcur/schedule"
	"blockConcur/types"

	types2 "github.com/ledgerwatch/erigon/core/types"
)

type FLAG int

const (
	START FLAG = iota
	END
)

type TaskMessage struct {
	Flag  FLAG
	Tasks types.Tasks
}

type BuildGraphMessage struct {
	Flag         FLAG
	Tasks        types.Tasks
	RwAccessedBy *rwset.RwAccessedBy
}

type GraphMessage struct {
	Flag  FLAG
	Graph *dag.Graph
}

type ScheduleMessage struct {
	Flag       FLAG
	Processors schedule.Processors
	Makespan   uint64
	Withdraws  types2.Withdrawals
}
