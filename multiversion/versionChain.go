package multiversion

import (
	"blockConcur/utils"
	"sync/atomic"
)

type VersionChain struct {
	Head       *Version
	LastCommit atomic.Value // only write-write conflicts, no read-write conflicts
	Tail       atomic.Value
}

func NewVersionChain(data interface{}) *VersionChain {
	head := NewVersion(data, utils.SnapshotID, Committed)
	atm := atomic.Value{}
	atm.Store(head)
	tail := atomic.Value{}
	tail.Store(head)
	return &VersionChain{
		Head:       head, // an dummy head which means its from the stateSnapshot
		Tail:       tail,
		LastCommit: atm, // the last committed version
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
	// CAS to ensure tail is always the true end of the chain
	for {
		tail := vc.Tail.Load().(*Version)
		if tail.Tid.Less(iv.Tid) {
			if vc.Tail.CompareAndSwap(tail, iv) {
				break
			}
		} else {
			// If the current tail is newer, we don't need to update
			break
		}
	}
}

func (vc *VersionChain) GetCommittedVersion() *Version {
	return vc.LastCommit.Load().(*Version)
}

// Find the last committed version and make it the new head
func (vc *VersionChain) GarbageCollection() *Version {
	cur_v := vc.GetCommittedVersion()
	vc.Head = cur_v
	return cur_v
}

func (vc *VersionChain) Prune(Tid *utils.ID) {
	for vc.Head != nil && vc.Head.Tid.Less(Tid) {
		vc.Head = vc.Head.Next
	}
	if vc.Head == nil {
		head := NewVersion(nil, utils.SnapshotID, Committed)
		vc.Head = head
		vc.Tail.Store(head)
		vc.LastCommit.Store(head)
	}
}

func (vc *VersionChain) GetLastBlockVersion(txid *utils.ID) *Version {
	cur := vc.Tail.Load().(*Version)
	for cur != nil && cur.Tid.BlockNumber >= txid.BlockNumber {
		cur = cur.Prev
	}
	if cur == nil {
		return vc.Head
	}
	return cur
}
