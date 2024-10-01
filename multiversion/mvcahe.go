package multiversion

import (
	innerstate "blockConcur/state/inner_state"
	"blockConcur/utils"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
)

// Support both read and write operations.
// We use version chain to store intermediate & committed states.
// stateCache stores only committed states, and is responsible for garbage collection.
// existCache is used to identify whether the address exists or not.
// snapshot is the intra-block state.
type MvCache struct {
	// vcChain: version chain per record: (addr || hash) -> *VersionChain
	// prize is stored at (COINBASE || PRIZE) -> *VersionChain
	vcCache  *lru.Cache[string, *VersionChain]
	snapshot *innerstate.IntraBlockState
	dirtyVc  map[string]struct{}
	prizeKey string
}

func NewMvCache(ibs *innerstate.IntraBlockState, cacheSize int) *MvCache {
	mvCache := &MvCache{
		snapshot: ibs,
	}

	cache, initErr := lru.NewWithEvict(cacheSize, func(key string, vc *VersionChain) {
		// it is actually a write-back process.
		// we add one bit to the entry to indicate whether it is dirty or not
		addr := common.BytesToAddress([]byte(key[:20]))
		hash := common.BytesToHash([]byte(key[20:]))
		commit_version := vc.getCommittedVersion()

		if commit_version.IsSnapshot() {
			value := commit_version.Data
			switch hash {
			case utils.BALANCE:
				ibs.SetBalance(addr, value.(*uint256.Int))
			case utils.NONCE:
				ibs.SetNonce(addr, value.(uint64))
			case utils.CODEHASH:
				ibs.SetCodeHash(addr, value.(common.Hash))
			case utils.CODE:
				ibs.SetCode(addr, value.([]byte))
			case utils.EXIST:
				if !value.(bool) {
					ibs.Selfdestruct(addr)
				}
			default:
				ibs.SetState(addr, &hash, value.(uint256.Int))
			}
		}
	})

	if initErr != nil {
		panic(initErr)
	}

	mvCache.vcCache = cache

	return mvCache
}

// new a version chain if not found, otherwise return the existing one.
// The newly-created version chain will be added to the cache.
func (mvc *MvCache) get_or_new_vc(key string) (*VersionChain, bool) {
	vc, ok := mvc.vcCache.Get(key)
	if !ok {
		vc = NewVersionChain()
		mvc.vcCache.Add(key, vc)
	}
	return vc, ok
}

// Set the prize key, which is used to store the prize for each transaction
func (mvc *MvCache) SetPrizeKey(coinbase common.Address) {
	mvc.prizeKey = utils.MakeKey(coinbase, utils.PRIZE)
}

// Insert a version into the mv_cache
func (mvs *MvCache) InsertVersion(key string, version *Version) {
	vc, _ := mvs.get_or_new_vc(key)
	vc.installVersion(version)
}

func (mvs *MvCache) GetCommittedVersion(key string) *Version {
	vc, _ := mvs.get_or_new_vc(key)
	return vc.getCommittedVersion()
}

// GC： only retain the last commit version of each chain
// GC is triggered by the end of each block
func (mvs *MvCache) GarbageCollection() {
	for key := range mvs.dirtyVc {
		vc, ok := mvs.vcCache.Get(key)
		if !ok {
			panic("version chain not found in the cache")
		}
		vc.garbageCollection()
	}
	mvs.dirtyVc = make(map[string]struct{})
}

// Fetch from the cache, if not found, fetch from the snapshot.
// This function will be called in 2 cases:
// 1. when the version is not in the read version chain of the transaction.
// 2. at the begining of the block, fetch the initial state from the snapshot.
// As now we are not considering inter-block concurrency, because the block state
// generation problem is also a big topic.
func (mvc *MvCache) Fetch(addr common.Address, hash common.Hash) interface{} {
	key := utils.MakeKey(addr, hash)
	vc, ok := mvc.get_or_new_vc(key)
	if ok {
		return vc.getCommittedVersion().Data
	}
	// fetch the data from the snapshot
	v := vc.getCommittedVersion()
	switch hash {
	case utils.BALANCE:
		v.Data = mvc.snapshot.GetBalance(addr)
	case utils.NONCE:
		v.Data = mvc.snapshot.GetNonce(addr)
	case utils.CODEHASH:
		v.Data = mvc.snapshot.GetCodeHash(addr)
	case utils.CODE:
		v.Data = mvc.snapshot.GetCode(addr)
	case utils.EXIST:
		v.Data = mvc.snapshot.Exist(addr)
	default:
		ret := uint256.NewInt(0)
		mvc.snapshot.GetState(addr, &hash, ret)
		v.Data = ret
	}
	return v.Data
}

// Upload an existing version to the version chain.
// This function may modify the version chain's last commit version.
// if v.status is committed and last_commit_version.Tid is less than v.Tid
// then last_commit_vereion = v. Using CAS here, and we will set the state cache.
func (mvc *MvCache) Update(v *Version, key string, value interface{}) {
	mvc.dirtyVc[key] = struct{}{}
	// Transaction waiting for this version to be committed can read this version
	v.Settle(Committed, value)
	vc, _ := mvc.get_or_new_vc(key)
	for {
		last_commit_version := vc.getCommittedVersion()
		if v.Tid.Less(last_commit_version.Tid) {
			return
		}
		if vc.LastCommit.CompareAndSwap(last_commit_version, v) {
			return
		}
	}
}

// go through the prize chain, add all committed versions to the result
// prize chain is initially as long as the length of transactions
func (mvc *MvCache) FetchPrize(TxId *utils.ID) *uint256.Int {
	prizeChain, ok := mvc.vcCache.Get(mvc.prizeKey)
	if !ok {
		panic("prize chain not found")
	}
	ret := uint256.NewInt(0)
	cur := prizeChain.Head
	for cur.Tid.Less(TxId) {
		if cur.Data != nil && cur.Status == Committed {
			if cur.Status == Pending {
				panic("pevious prize is pending, this transaction should not be executed")
			}
			ret = ret.Add(ret, cur.Data.(*uint256.Int))
		}
		cur = cur.Next
	}
	return ret
}

// TODO: if we want to run continously, we need to prune the prize chain in the end of each block.
// also collect the prize to the coinbase.
func (mvc *MvCache) PrunePrize(TxId *utils.ID) {
	prizeChain, ok := mvc.vcCache.Get(mvc.prizeKey)
	if !ok {
		panic("prize chain not found")
	}
	prizeChain.prune(TxId)
}
