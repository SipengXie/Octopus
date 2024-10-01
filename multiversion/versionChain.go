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

func (vc *VersionChain) installVersion(iv *Version) {
	cur_v := vc.Head
	for {
		if cur_v == nil {
			break
		}
		cur_v = cur_v.insertOrNext(iv)
	}
}

func (vc *VersionChain) getCommittedVersion() *Version {
	return vc.LastCommit.Load().(*Version)
}

// Find the last committed version
func (vc *VersionChain) garbageCollection() *Version {

	cur_v := vc.getCommittedVersion()
	newHead := NewVersion(nil, utils.SnapshotID, Committed)
	if cur_v != nil {
		newHead.Data = cur_v.Data
	}
	vc.Head = newHead
	vc.LastCommit.Store(newHead)

	return newHead
}

func (vc *VersionChain) prune(Tid *utils.ID) {
	if Tid == utils.EndID {
		// reset the version chain
		vc.Head = NewVersion(nil, utils.SnapshotID, Committed)
		vc.LastCommit.Store(vc.Head)
		return
	}
	cur := vc.Head
	for cur.Tid.Less(Tid) {
		cur = cur.Next
	}
	vc.Head = cur
}
