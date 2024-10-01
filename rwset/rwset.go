package rwset

import (
	"blockConcur/utils"
	"fmt"

	"github.com/ledgerwatch/erigon-lib/common"
)

type accessMap map[string]struct{}

type RWSetList []*RwSet

func (tuple accessMap) Add(addr common.Address, hash common.Hash) {
	key := utils.MakeKey(addr, hash)
	if _, ok := tuple[key]; !ok {
		tuple[key] = struct{}{}
	}
}

func (tuple accessMap) Contains(addr common.Address, hash common.Hash) bool {
	key := utils.MakeKey(addr, hash)
	_, ok := tuple[key]
	return ok
}

type RwSet struct {
	ReadSet  accessMap
	WriteSet accessMap
}

func NewRwSet() *RwSet {
	return &RwSet{
		ReadSet:  make(accessMap),
		WriteSet: make(accessMap),
	}
}

func (RWSets *RwSet) AddReadSet(addr common.Address, hash common.Hash) {
	if RWSets == nil {
		fmt.Println("NewRWSets is nil")
		return
	}
	RWSets.ReadSet.Add(addr, hash)
}

func (RWSets *RwSet) AddWriteSet(addr common.Address, hash common.Hash) {
	if RWSets == nil {
		fmt.Println("NewRWSets is nil")
		return
	}
	RWSets.WriteSet.Add(addr, hash)
}

func (RWSets *RwSet) Equal(other *RwSet) bool {
	if len(RWSets.ReadSet) != len(other.ReadSet) {
		return false
	}
	if len(RWSets.WriteSet) != len(other.WriteSet) {
		return false
	}

	for key := range RWSets.ReadSet {
		if _, ok := other.ReadSet[key]; !ok {
			return false
		}
	}

	for key := range RWSets.WriteSet {
		if _, ok := other.WriteSet[key]; !ok {
			return false
		}
	}

	return true
}
