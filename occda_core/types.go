package occda_core

import (
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
)

type OCCDATask struct {
	types.Task
	sid           *utils.ID
	gasUsed       uint64
	stateToCommit *state.ExecState
}

func NewOCCDATask(task *types.Task, sid *utils.ID) *OCCDATask {
	ret := &OCCDATask{
		Task: *task,
		sid:  sid,
	}
	ret.RwSet = nil
	ret.ReadVersions = nil
	ret.WriteVersions = nil
	ret.PrizeVersions = nil
	return ret
}

func GenerateOCCDATasks(tasks []*types.Task) []*OCCDATask {
	ret := make([]*OCCDATask, len(tasks))
	for i, task := range tasks {
		ret[i] = NewOCCDATask(task, utils.SnapshotID)
	}
	return ret
}

// if t's read set has overlap with other's write set, then t depends on other
func (t *OCCDATask) Depend(other *OCCDATask) bool {
	readset := t.RwSet.ReadSet
	writeset := other.RwSet.WriteSet
	for key := range readset {
		if _, ok := writeset[key]; ok {
			return true
		}
	}
	return false
}
