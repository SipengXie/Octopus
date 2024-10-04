package types

import (
	mv "blockConcur/multiversion"
	"blockConcur/rwset"
	"blockConcur/utils"

	"github.com/ledgerwatch/erigon-lib/common"
	types2 "github.com/ledgerwatch/erigon/core/types"
)

type Task struct {
	Tid   *utils.ID
	Cost  uint64
	Msg   *types2.Message
	RwSet *rwset.RwSet

	BlockHash common.Hash
	TxHash    common.Hash

	ReadVersions  map[string]*mv.Version
	WriteVersions map[string]*mv.Version
	PrizeVersions []*mv.Version
}

func NewTask(id *utils.ID, cost uint64, msg *types2.Message, bHash, tHash common.Hash) *Task {
	return &Task{
		Tid:           id,
		Cost:          cost,
		Msg:           msg,
		BlockHash:     bHash,
		TxHash:        tHash,
		RwSet:         nil,
		ReadVersions:  make(map[string]*mv.Version),
		WriteVersions: make(map[string]*mv.Version),
	}
}

func (t *Task) AddReadVersion(key string, version *mv.Version) {
	t.ReadVersions[key] = version
}

func (t *Task) AddWriteVersion(key string, version *mv.Version) {
	t.WriteVersions[key] = version
}

func (t *Task) AddPrizeVersion(version *mv.Version) {
	t.PrizeVersions = append(t.PrizeVersions, version)
}

func (t *Task) Wait() {
	for _, version := range t.ReadVersions {
		version.Mu.Lock()
		for version.Status == mv.Pending {
			version.Cond.Wait()
		}
		version.Mu.Unlock()
	}
	for _, version := range t.PrizeVersions {
		version.Mu.Lock()
		for version.Status == mv.Pending {
			version.Cond.Wait()
		}
		version.Mu.Unlock()
	}
}

// we assume Tasks are sorted by GlobalId
type Tasks []*Task

func (t Tasks) Len() int {
	return len(t)
}

func (t Tasks) Less(i, j int) bool {
	return t[i].Tid.Less(t[j].Tid)
}

func (t Tasks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// use binary search to find the target task.
// if not found, we find the first task that is less than the target.
func (t Tasks) Find(target *utils.ID) (int, bool) {
	left, right := 0, len(t)-1

	for left <= right {
		mid := (left + right) / 2
		if t[mid].Tid.Equal(target) {
			return mid, true
		} else if t[mid].Tid.Less(target) {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return right, false
}
