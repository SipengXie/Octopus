package rwset

import (
	"sort"

	"github.com/ledgerwatch/erigon-lib/common"
)

// readBy / writeBy
type AccessedBy map[common.Address]map[common.Hash]map[int]struct{}

func NewAccessedBy() AccessedBy {
	return make(map[common.Address]map[common.Hash]map[int]struct{})
}

func (accessedBy AccessedBy) Add(addr common.Address, hash common.Hash, txID int) {
	if _, ok := accessedBy[addr]; !ok {
		accessedBy[addr] = make(map[common.Hash]map[int]struct{})
	}
	if _, ok := accessedBy[addr][hash]; !ok {
		accessedBy[addr][hash] = make(map[int]struct{})
	}
	accessedBy[addr][hash][txID] = struct{}{}
}

func (accessedBy AccessedBy) TxIds(addr common.Address, hash common.Hash) []int {
	txIds := make([]int, 0)

	if _, ok := accessedBy[addr]; !ok {
		return txIds
	} else if _, ok := accessedBy[addr][hash]; !ok {
		return txIds
	} else {
		for txID := range accessedBy[addr][hash] {
			txIds = append(txIds, txID)
		}
	}

	sort.Slice(txIds, func(i, j int) bool {
		return txIds[i] < txIds[j]
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

func (rw *RwAccessedBy) Add(set *RWSet, txId int) {
	if set == nil {
		return
	}
	for addr, state := range set.ReadSet {
		for hash := range state {
			rw.ReadBy.Add(addr, hash, txId)
		}
	}
	for addr, state := range set.WriteSet {
		for hash := range state {
			rw.WriteBy.Add(addr, hash, txId)
		}
	}
}

func (rw *RwAccessedBy) Copy() *RwAccessedBy {
	newRw := NewRwAccessedBy()
	for addr, hashMap := range rw.ReadBy {
		for hash, txMap := range hashMap {
			for txId := range txMap {
				newRw.ReadBy.Add(addr, hash, txId)
			}
		}
	}
	for addr, hashMap := range rw.WriteBy {
		for hash, txMap := range hashMap {
			for txId := range txMap {
				newRw.WriteBy.Add(addr, hash, txId)
			}
		}
	}
	return newRw
}
