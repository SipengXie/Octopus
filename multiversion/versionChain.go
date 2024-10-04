package multiversion

import (
	"blockConcur/utils"
	"sync/atomic"
)

type VersionChain struct {
	Head       *Version
	LastCommit atomic.Value // only write-write conflicts, no read-write conflicts
}

func NewVersionChain() *VersionChain {
	head := NewVersion(nil, utils.SnapshotID, Committed)
	atm := atomic.Value{}
	atm.Store(head)
	return &VersionChain{
		Head:       head, // an dummy head which means its from the stateSnapshot
		LastCommit: atm,  // the last committed version
	}
}

func (vc *VersionChain) InstallVersion(iv *Version) {
	cur_v := vc.Head
	for {
		if cur_v == nil {
			break
		}
		cur_v = cur_v.insertOrNext(iv)
	}
}

func (vc *VersionChain) GetCommittedVersion() *Version {
	return vc.LastCommit.Load().(*Version)
}

// Find the last committed version
func (vc *VersionChain) GarbageCollection() *Version {
	cur_v := vc.GetCommittedVersion()
	// newHead := NewVersion(cur_v.Data, utils.SnapshotID, Committed)
	// vc.Head = newHead
	// vc.LastCommit.Store(newHead)
	vc.Head = cur_v

	return cur_v
}

func (vc *VersionChain) Prune(Tid *utils.ID) {
	cur := vc.Head
	for cur.Tid.Less(Tid) {
		cur = cur.Next
	}
	vc.Head = cur
}
