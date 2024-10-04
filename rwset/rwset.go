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

func (tuple accessMap) Print() {
	output := make(map[string][]string)
	for key := range tuple {
		addr, hash := utils.ParseKey(key)
		if list, ok := output[addr.Hex()]; !ok {
			output[addr.Hex()] = []string{utils.DecodeHash(hash)}
		} else {
			list = append(list, utils.DecodeHash(hash))
			output[addr.Hex()] = list
		}
	}
	for addr, hashList := range output {
		fmt.Printf("\t%s:\n", addr)
		for _, hash := range hashList {
			fmt.Printf("\t\t%s\n", hash)
		}
	}
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

func (set *RwSet) BasicRwSet(sender, to common.Address, is_transfer, is_call, is_coinbase bool) {
	set.AddReadSet(sender, utils.NONCE)
	set.AddWriteSet(sender, utils.NONCE)

	set.AddWritePrize()
	if is_coinbase {
		set.AddReadPrize()
	}

	if is_transfer {
		set.AddReadSet(sender, utils.BALANCE)
		set.AddWriteSet(sender, utils.BALANCE)
		if to != (common.Address{}) { // not create
			set.AddReadSet(to, utils.BALANCE)
			set.AddWriteSet(to, utils.BALANCE)
		}
	}

	if is_call {
		if to != (common.Address{}) { // not create
			set.AddReadSet(to, utils.CODE)
			set.AddReadSet(to, utils.CODEHASH)
		}
	}
}

func (set *RwSet) AddReadSet(addr common.Address, hash common.Hash) {
	if set == nil {
		fmt.Println("NewRWSets is nil")
		return
	}
	set.ReadSet.Add(addr, hash)
}

func (RWSet *RwSet) AddReadPrize() {
	RWSet.ReadSet["prize"] = struct{}{}
}

func (RWSets *RwSet) AddWriteSet(addr common.Address, hash common.Hash) {
	if RWSets == nil {
		fmt.Println("NewRWSets is nil")
		return
	}
	RWSets.WriteSet.Add(addr, hash)
}

func (RWSet *RwSet) AddWritePrize() {
	RWSet.WriteSet["prize"] = struct{}{}
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
