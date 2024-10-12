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
	Flag      FLAG
	Tasks     types.Tasks
	PostBlock *types.Task
	Header    *types2.Header
	Headers   []*types2.Header
	Withdraws types2.Withdrawals
}

type BuildGraphMessage struct {
	Flag         FLAG
	Tasks        types.Tasks
	PostBlock    *types.Task
	RwAccessedBy *rwset.RwAccessedBy
	Header       *types2.Header
	Headers      []*types2.Header
	Withdraws    types2.Withdrawals
}

type GraphMessage struct {
	Flag      FLAG
	Graph     *dag.Graph
	PostBlock *types.Task
	Header    *types2.Header
	Headers   []*types2.Header
	Withdraws types2.Withdrawals
}

type ScheduleMessage struct {
	Flag       FLAG
	Processors schedule.Processors
	Makespan   uint64
	PostBlock  *types.Task
	Withdraws  types2.Withdrawals
	Header     *types2.Header
	Headers    []*types2.Header
}
