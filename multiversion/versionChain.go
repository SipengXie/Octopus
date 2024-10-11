package multiversion

import (
	"blockConcur/utils"
	"sync/atomic"
)

type VersionChain struct {
	Head            *Version
	LastCommit      atomic.Value // only write-write conflicts, no read-write conflicts
	LastBlockCommit *Version
}

func NewVersionChain(blockNumber uint64) *VersionChain {
	head := NewVersion(nil, utils.NewID(blockNumber, -1, 0), Committed)
	atm := atomic.Value{}
	atm.Store(head)
	return &VersionChain{
		Head:            head, // an dummy head which means its from the stateSnapshot
		LastBlockCommit: head,
		LastCommit:      atm, // the last committed version
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

// Find the last committed version and put it into a snapshot
func (vc *VersionChain) GarbageCollection() *Version {
	cur_v := vc.GetCommittedVersion()
	vc.LastBlockCommit = cur_v
	vc.Head = cur_v
	return cur_v
}

func (vc *VersionChain) Prune(Tid *utils.ID) {
	for vc.Head != nil && vc.Head.Tid.Less(Tid) {
		vc.Head = vc.Head.Next
	}
	if vc.Head == nil {
		vc.Head = NewVersion(nil, utils.NewID(Tid.BlockNumber+1, -1, 0), Committed)
		vc.LastCommit.Store(vc.Head)
	}
}
