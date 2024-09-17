package multiversion

type VersionChain struct {
	Head          *Version
	Tail          *Version
	LastBlockTail *Version
}

func NewVersionChain() *VersionChain {
	head := NewVersion(nil, SnapshotID, Committed)
	return &VersionChain{
		Head:          head, // an dummy head which means its from the stateSnapshot
		Tail:          head,
		LastBlockTail: head,
	}
}

func (vc *VersionChain) InstallVersion(iv *Version) {
	cur_v := vc.Head
	for {
		if cur_v == nil {
			break
		}
		cur_v = cur_v.InsertOrNext(iv)
	}
	if iv.Next == nil {
		vc.Tail = iv
	}
}

func (vc *VersionChain) UpdateLastBlockTail() {
	vc.LastBlockTail = vc.Tail
}

// Find the last committed version
func (vc *VersionChain) GarbageCollection() *Version {

	cur_v := vc.Tail
	for {
		if cur_v == nil {
			break
		}
		if cur_v.Status == Committed {
			break
		}
		cur_v = cur_v.Prev
	}

	newHead := NewVersion(nil, SnapshotID, Committed)
	if cur_v != nil {
		newHead.Data = cur_v.Data
	}
	vc.Head = newHead
	vc.Tail = newHead
	vc.LastBlockTail = newHead

	return newHead
}
