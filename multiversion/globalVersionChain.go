package multiversion

import (
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"sync"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
)

type GlobalVersionChain struct {
	ChainMap   sync.Map // ChainMap: version chain per record: addr -> hash -> *VersionChain
	innerState state.ColdState
}

func NewGlobalVersionChain(ibs state.ColdState) *GlobalVersionChain {
	return &GlobalVersionChain{
		ChainMap:   sync.Map{},
		innerState: ibs,
	}
}

// ------------------ Insert Version ---------------------

// hash : BALANCE, NONCE, CODE, CODEHASH, ALIVE, SLOTS
func (mvs *GlobalVersionChain) InsertVersion(addr common.Address, hash common.Hash, version *Version) {
	cache, _ := mvs.ChainMap.LoadOrStore(addr, &sync.Map{})
	vc, _ := cache.(*sync.Map).LoadOrStore(hash, NewVersionChain())
	vc.(*VersionChain).InstallVersion(version)
}

// -------------------- Get LastBlockTail Version --------------------

// hash : BALANCE, NONCE, CODE, CODEHASH, ALIVE, SLOTS
func (mvs *GlobalVersionChain) GetLastBlockTailVersion(addr common.Address, hash common.Hash) *Version {
	cache, _ := mvs.ChainMap.LoadOrStore(addr, &sync.Map{})
	vc, _ := cache.(*sync.Map).LoadOrStore(hash, NewVersionChain())
	return vc.(*VersionChain).LastBlockTail
}

func writeBack(v *Version, innerState state.ColdState, addr common.Address, hash common.Hash) {
	// switch hash {
	// case rwset.BALANCE:
	// 	innerState.SetBalance(addr, v.Data.(*uint256.Int))
	// case rwset.NONCE:
	// 	innerState.SetNonce(addr, v.Data.(uint64))
	// case rwset.CODE:
	// 	innerState.SetCode(addr, v.Data.([]byte))
	// case rwset.CODEHASH:
	// 	// innerState.SetCodeHash(addr, v.Data.(common.Hash))
	// case rwset.ALIVE:
	// 	if !v.Data.(bool) {
	// 		innerState.Selfdestruct(addr)
	// 	}
	// default:
	// 	innerState.SetState(addr, &hash, *v.Data.(*uint256.Int))
	// }
}

func (mvs *GlobalVersionChain) GarbageCollection() {
	mvs.ChainMap.Range(func(key, value interface{}) bool {
		addr := key.(common.Address)
		cache := value.(*sync.Map)
		cache.Range(func(key, value interface{}) bool {
			hash := key.(common.Hash)
			vc := value.(*VersionChain)
			newhead := vc.GarbageCollection()
			writeBack(newhead, mvs.innerState, addr, hash)
			return true
		})
		return true
	})
}

func setVersion(v *Version, innerState state.ColdState, addr common.Address, hash common.Hash) {
	switch hash {
	case rwset.BALANCE:
		v.Data = innerState.GetBalance(addr)
	case rwset.NONCE:
		v.Data = innerState.GetNonce(addr)
	case rwset.CODE:
		v.Data = innerState.GetCode(addr)
	case rwset.CODEHASH:
		v.Data = innerState.GetCodeHash(addr)
	case rwset.ALIVE:
		v.Data = !innerState.HasSelfdestructed(addr)
	default:
		ret := uint256.NewInt(0)
		innerState.GetState(addr, &hash, ret)
		v.Data = ret
	}
}

func (gvc *GlobalVersionChain) DoPrefetch(addr common.Address, hash common.Hash) {
	v := gvc.GetLastBlockTailVersion(addr, hash)
	if v.Data != nil || v.Tid != types.SnapshotID {
		return
	}
	setVersion(v, gvc.innerState, addr, hash)
}

func (gvc *GlobalVersionChain) UpdateLastBlockTail() {
	gvc.ChainMap.Range(func(key, value interface{}) bool {
		// addr := key.(common.Address)
		cache := value.(*sync.Map)
		cache.Range(func(key, value interface{}) bool {
			// hash := key.(common.Hash)
			vc := value.(*VersionChain)
			vc.UpdateLastBlockTail()
			return true
		})
		return true
	})
}
