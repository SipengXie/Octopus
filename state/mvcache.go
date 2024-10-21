package state

import (
	mv "blockConcur/multiversion"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"reflect"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
)

type snapshotInterface interface {
	GetBalance(addr common.Address) *uint256.Int
	GetNonce(addr common.Address) uint64
	GetCodeHash(addr common.Address) common.Hash
	GetCode(addr common.Address) []byte
	Exist(addr common.Address) bool
	GetState(addr common.Address, hash *common.Hash, ret *uint256.Int)
	Selfdestruct(addr common.Address) bool
	SetBalance(addr common.Address, value *uint256.Int)
	SetNonce(addr common.Address, value uint64)
	SetCodeHash(addr common.Address, value common.Hash)
	SetCode(addr common.Address, value []byte)
	SetState(addr common.Address, hash *common.Hash, value uint256.Int)
	CreateAccount(addr common.Address)
}

// Support both read and write operations.
// We use version chain to store intermediate & committed states.
// stateCache stores only committed states, and is responsible for garbage collection.
// existCache is used to identify whether the address exists or not.
// snapshot is the intra-block state.
type MvCache struct {
	// vcChain: version chain per record: (addr || hash) -> *VersionChain
	// prize is stored at (COINBASE || PRIZE) -> *VersionChain
	vcCache    *lru.Cache[string, *mv.VersionChain]
	prizeChain *mv.VersionChain
	snapshot   snapshotInterface
	dirtyVc    sync.Map
	coinbase   common.Address
}

func NewMvCache(ibs *IntraBlockState, cacheSize int) *MvCache {
	mvCache := &MvCache{
		prizeChain: mv.NewVersionChain(),
		dirtyVc:    sync.Map{},
	}
	snapshot := NewFakeInnerState(ibs)
	cache, initErr := lru.NewWithEvict(cacheSize, func(key string, vc *mv.VersionChain) {
		// it is actually a write-back process.
		addr := common.BytesToAddress([]byte(key[:20]))
		hash := common.BytesToHash([]byte(key[20:]))
		commit_version := vc.GetCommittedVersion()

		if !commit_version.IsSnapshot() {
			value := commit_version.Data
			switch hash {
			case utils.BALANCE:
				snapshot.SetBalance(addr, value.(*uint256.Int))
			case utils.NONCE:
				snapshot.SetNonce(addr, value.(uint64))
			case utils.CODEHASH:
				snapshot.SetCodeHash(addr, value.(common.Hash))
			case utils.CODE:
				snapshot.SetCode(addr, value.([]byte))
			case utils.EXIST:
				if !value.(bool) {
					snapshot.Selfdestruct(addr)
				} else {
					snapshot.CreateAccount(addr)
				}
			default:
				snapshot.SetState(addr, &hash, *value.(*uint256.Int))
			}
		}
	})

	if initErr != nil {
		panic(initErr)
	}

	mvCache.vcCache = cache
	mvCache.snapshot = snapshot

	return mvCache
}

// new a version chain if not found, otherwise return the existing one.
// The newly-created version chain will be added to the cache.
func (mvc *MvCache) get_or_new_vc(key string) (*mv.VersionChain, bool) {
	vc, ok := mvc.vcCache.Get(key)
	if !ok {
		vc = mv.NewVersionChain()
		mvc.vcCache.Add(key, vc)
	}
	return vc, ok
}

// Set the prize key, which is used to store the prize for each transaction
func (mvc *MvCache) SetCoinbase(coinbase common.Address) {
	mvc.coinbase = coinbase
}

// Insert a version into the mv_cache
func (mvs *MvCache) InsertVersion(key string, version *mv.Version) {
	if key == "prize" {
		mvs.prizeChain.InstallVersion(version)
		return
	}
	vc, _ := mvs.get_or_new_vc(key)
	vc.InstallVersion(version)
}

func (mvs *MvCache) GetLastBlockVersion(key string, txid *utils.ID) *mv.Version {
	if key == "prize" {
		return mvs.prizeChain.GetLastBlockVersion(txid)
	}
	vc, _ := mvs.get_or_new_vc(key)
	return vc.GetLastBlockVersion(txid)
}

func (mvs *MvCache) gcForSerial(balanceUpdate map[common.Address]*uint256.Int, txid *utils.ID) {
	for addr, balanceChange := range balanceUpdate {
		oldBalance := mvs.Fetch(addr, utils.BALANCE).(*uint256.Int)
		newBalance := new(uint256.Int).Add(oldBalance, balanceChange)
		// update balance
		version := mv.NewVersion(newBalance, txid, mv.Committed)
		mvs.InsertVersion(utils.MakeKey(addr, utils.BALANCE), version)
		mvs.Update(version, utils.MakeKey(addr, utils.BALANCE), newBalance)
		// TODO: maybe we will update exist
		version = mv.NewVersion(true, txid, mv.Committed)
		mvs.InsertVersion(utils.MakeKey(addr, utils.EXIST), version)
		mvs.Update(version, utils.MakeKey(addr, utils.EXIST), true)
	}
}

// GC： only retain the last commit version of each chain
// GC is triggered by the end of each block
// fetch the prize and add to the coinbase
func (mvs *MvCache) GarbageCollection(balanceUpdate map[common.Address]*uint256.Int, post_block_task *types.Task) {
	// Combine balance update and prize collection into a single operation
	txId := post_block_task.Tid
	// Fetch and add prize to coinbase's balance update
	prize := mvs.FetchPrize(txId)
	if !prize.IsZero() {
		if balanceUpdate == nil {
			balanceUpdate = make(map[common.Address]*uint256.Int)
		}
		if _, exists := balanceUpdate[mvs.coinbase]; !exists {
			balanceUpdate[mvs.coinbase] = new(uint256.Int)
		}
		balanceUpdate[mvs.coinbase].Add(balanceUpdate[mvs.coinbase], prize)
	}

	// Apply all balance updates with the same txId
	if len(balanceUpdate) > 0 {
		post_block_write_versions := post_block_task.WriteVersions
		if len(post_block_write_versions) == 0 {
			mvs.gcForSerial(balanceUpdate, txId)
		} else {
			// update post_block_write_versions if the corresponding account is in the map
			for key, version := range post_block_write_versions {
				addr, hash := utils.ParseKey(key)
				// TODO: may be we would update exist
				if hash == utils.EXIST {
					mvs.Update(version, key, true)
				} else if balanceChange, exists := balanceUpdate[addr]; exists && !balanceChange.IsZero() {
					oldBalance := mvs.Fetch(addr, utils.BALANCE).(*uint256.Int)
					newBalance := new(uint256.Int).Add(oldBalance, balanceChange)
					mvs.Update(version, key, newBalance)
				} else {
					version.Settle(mv.Ignore, nil)
				}
			}
		}
	}

	mvs.PrunePrize(txId)
	mvs.dirtyVc.Range(func(key, _ any) bool {
		vc, ok := mvs.vcCache.Get(key.(string))
		if !ok {
			return true
			// addr, hash := utils.ParseKey(key.(string))
			// panic("version chain not found in the cache: " + addr.Hex() + " " + hash.Hex())
		}
		vc.GarbageCollection()
		return true
	})
	mvs.dirtyVc = sync.Map{}
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
	if ok && vc.GetCommittedVersion().Data != nil {
		return vc.GetCommittedVersion().Data
	}
	// fetch the data from the snapshot
	v := vc.GetCommittedVersion()
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

func (mvc *MvCache) peekFetch(key string) *mv.Version {
	vc, ok := mvc.vcCache.Peek(key)
	if !ok {
		return nil
	}
	lastCommit := vc.GetCommittedVersion()
	return lastCommit
}

func (mvc *MvCache) peekExist(addr common.Address) bool {
	v := mvc.peekFetch(utils.MakeKey(addr, utils.EXIST))
	if v != nil {
		return v.Data.(bool)
	}
	return mvc.snapshot.Exist(addr)
}

// TODO: There're mistakes such as selfdestructed6780
func (mvc *MvCache) Validate(ibs *IntraBlockState) *utils.ID {
	minTid := utils.EndID
	keys := mvc.vcCache.Keys()
	for _, key := range keys {
		lastCommit := mvc.peekFetch(key)
		if lastCommit.IsSnapshot() {
			continue
		}
		val := lastCommit.Data
		addr, hash := utils.ParseKey(key)
		if lastCommit.Tid.TxIndex == 114 && lastCommit.Tid.BlockNumber == 19452469 && hash == utils.EXIST {
			fmt.Println(lastCommit.Tid)
		}
		var ibsValue interface{}
		switch hash {
		case utils.BALANCE:
			ibsValue = ibs.GetBalance(addr)
		case utils.NONCE:
			ibsValue = ibs.GetNonce(addr)
		case utils.CODEHASH:
			ibsValue = ibs.GetCodeHash(addr)
		case utils.CODE:
			ibsValue = ibs.GetCode(addr)
		case utils.EXIST:
			ibsValue = ibs.Exist(addr)
		default:
			var stateValue uint256.Int
			ibs.GetState(addr, &hash, &stateValue)
			ibsValue = &stateValue
		}
		if hash != utils.EXIST {
			is_exist := mvc.peekExist(addr)
			is_exist_ibs := ibs.Exist(addr)
			if is_exist != is_exist_ibs {
				if lastCommit.Tid.Less(minTid) {
					minTid = lastCommit.Tid
				}
			} else if !is_exist {
				continue
			}
		}
		if !reflect.DeepEqual(ibsValue, val) {
			if lastCommit.Tid.Less(minTid) {
				minTid = lastCommit.Tid
			}
		}
	}

	if minTid == utils.EndID {
		return nil
	}
	return minTid
}

// Upload an existing version to the version chain.
// This function may modify the version chain's last commit version.
// if v.status is committed and last_commit_version.Tid is less than v.Tid
// then last_commit_vereion = v. Using CAS here, and we will set the state cache.
func (mvc *MvCache) Update(v *mv.Version, key string, value interface{}) {
	mvc.dirtyVc.Store(key, struct{}{})
	// Transaction waiting for this version to be committed can read this version
	v.Settle(mv.Committed, value)
	vc, _ := mvc.get_or_new_vc(key)
	for {
		last_commit_version := vc.GetCommittedVersion()
		if v.Tid.Less(last_commit_version.Tid) {
			return
		}
		if vc.LastCommit.CompareAndSwap(last_commit_version, v) {
			return
		}
	}
}

func (mvc *MvCache) UpdatePrize(v *mv.Version, value interface{}) {
	v.Settle(mv.Committed, value)
	vc := mvc.prizeChain
	for {
		last_commit_version := vc.GetCommittedVersion()
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
	ret := uint256.NewInt(0)
	cur := mvc.prizeChain.Head
	for cur != nil && cur.Tid.Less(TxId) {
		cur.Wait()
		// if cur.Status == mv.Pending {
		// 	panic("pevious prize is pending, this transaction should not be executed")
		// }
		if cur.Data != nil && cur.Status == mv.Committed {
			ret = ret.Add(ret, cur.Data.(*uint256.Int))
		}
		cur = cur.Next
	}
	return ret
}

// TODO: if we want to run continously, we need to prune the prize chain in the end of each block.
// also collect the prize to the coinbase.
func (mvc *MvCache) PrunePrize(TxId *utils.ID) {
	mvc.prizeChain.Prune(TxId)
}
