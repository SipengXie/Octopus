package rwset

import (
	"octopus/utils"
	"sort"
)

// readBy / writeBy
type AccessedBy map[string]utils.IDs

func NewAccessedBy() AccessedBy {
	return make(map[string]utils.IDs)
}

func (accessedBy AccessedBy) Add(key string, txID *utils.ID) {
	if _, ok := accessedBy[key]; !ok {
		accessedBy[key] = make(utils.IDs, 0)
	}
	accessedBy[key] = append(accessedBy[key], txID)
}

func (accessedBy AccessedBy) TxIds(key string) utils.IDs {
	txIds := accessedBy[key]
	sort.Slice(txIds, func(i, j int) bool {
		return (txIds[i]).Less((txIds[j]))
	})
	return txIds
}

type RwAccessedBy struct {
	ReadBy  AccessedBy
	WriteBy AccessedBy
}

func NewRwAccessedBy() *RwAccessedBy {
	return &RwAccessedBy{
		ReadBy:  NewAccessedBy(),
		WriteBy: NewAccessedBy(),
	}
}

func (rw *RwAccessedBy) Add(set *RwSet, txId *utils.ID) {
	if set == nil {
		return
	}

	for key := range set.ReadSet {
		rw.ReadBy.Add(key, txId)
	}

	for key := range set.WriteSet {
		rw.WriteBy.Add(key, txId)
	}
}
