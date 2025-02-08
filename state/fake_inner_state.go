package state

import (
	"octopus/utils"

	"github.com/alphadose/haxmap"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
)

type FakeInnerState struct {
	ibs     *IntraBlockState
	storage *haxmap.Map[string, interface{}]
}

func NewFakeInnerState(ibs *IntraBlockState) *FakeInnerState {
	return &FakeInnerState{ibs: ibs, storage: haxmap.New[string, interface{}]()}
}

func (f *FakeInnerState) GetBalance(addr common.Address) *uint256.Int {
	key := utils.MakeKey(addr, utils.BALANCE)
	if val, ok := f.storage.Get(key); ok {
		return val.(*uint256.Int)
	}
	return f.ibs.GetBalance(addr)
}

func (f *FakeInnerState) GetNonce(addr common.Address) uint64 {
	key := utils.MakeKey(addr, utils.NONCE)
	if val, ok := f.storage.Get(key); ok {
		return val.(uint64)
	}
	return f.ibs.GetNonce(addr)
}

func (f *FakeInnerState) GetCodeHash(addr common.Address) common.Hash {
	key := utils.MakeKey(addr, utils.CODEHASH)
	if val, ok := f.storage.Get(key); ok {
		return val.(common.Hash)
	}
	return f.ibs.GetCodeHash(addr)
}

func (f *FakeInnerState) GetCode(addr common.Address) []byte {
	key := utils.MakeKey(addr, utils.CODE)
	if val, ok := f.storage.Get(key); ok {
		return val.([]byte)
	}
	return f.ibs.GetCode(addr)
}

func (f *FakeInnerState) Exist(addr common.Address) bool {
	key := utils.MakeKey(addr, utils.EXIST)
	if val, ok := f.storage.Get(key); ok {
		return val.(bool)
	}
	return f.ibs.Exist(addr)
}

func (f *FakeInnerState) GetState(addr common.Address, hash *common.Hash, ret *uint256.Int) {
	key := utils.MakeKey(addr, *hash)
	if val, ok := f.storage.Get(key); ok {
		*ret = *val.(*uint256.Int)
	} else {
		f.ibs.GetState(addr, hash, ret)
	}
}

func (f *FakeInnerState) Selfdestruct(addr common.Address) bool {
	key := utils.MakeKey(addr, utils.EXIST)
	f.storage.Set(key, false)
	return true
}

func (f *FakeInnerState) SetBalance(addr common.Address, value *uint256.Int) {
	key := utils.MakeKey(addr, utils.BALANCE)
	f.storage.Set(key, value)
}

func (f *FakeInnerState) SetNonce(addr common.Address, value uint64) {
	key := utils.MakeKey(addr, utils.NONCE)
	f.storage.Set(key, value)
}

func (f *FakeInnerState) SetCodeHash(addr common.Address, value common.Hash) {
	key := utils.MakeKey(addr, utils.CODEHASH)
	f.storage.Set(key, value)
}

func (f *FakeInnerState) SetCode(addr common.Address, value []byte) {
	key := utils.MakeKey(addr, utils.CODE)
	f.storage.Set(key, value)
}

func (f *FakeInnerState) SetState(addr common.Address, hash *common.Hash, value uint256.Int) {
	key := utils.MakeKey(addr, *hash)
	f.storage.Set(key, &value)
}

func (f *FakeInnerState) CreateAccount(addr common.Address) {
	key := utils.MakeKey(addr, utils.EXIST)
	f.storage.Set(key, true)
}
