package state

import (
	"blockConcur/utils"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
)

type localWrite struct {
	storage map[common.Address]map[common.Hash]interface{}

	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logs         []*types.Log
	logSize      uint

	prize *uint256.Int
}

func newLocalWrite() *localWrite {
	return &localWrite{
		storage: make(map[common.Address]map[common.Hash]interface{}),
		logs:    make([]*types.Log, 0),
		refund:  0,

		prize: uint256.NewInt(0),
	}
}

// --------------------- Getters ------------------------------

func (lw *localWrite) getBalance(addr common.Address) (*uint256.Int, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return nil, false
	}
	if val, ok := lw.storage[addr][utils.BALANCE]; ok {
		return val.(*uint256.Int), true
	}
	return nil, false
}

func (lw *localWrite) getNonce(addr common.Address) (uint64, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return 0, false
	}
	if val, ok := lw.storage[addr][utils.NONCE]; ok {
		return val.(uint64), true
	}
	return 0, false
}

func (lw *localWrite) getCode(addr common.Address) ([]byte, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return nil, false
	}
	if code, ok := lw.storage[addr][utils.CODE]; ok {
		return code.([]byte), true
	}
	return nil, false
}

func (lw *localWrite) getCodeHash(addr common.Address) (common.Hash, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return common.Hash{}, false
	}
	if val, ok := lw.storage[addr][utils.CODEHASH]; ok {
		return val.(common.Hash), true
	}
	return common.Hash{}, false
}

func (lw *localWrite) getSlot(addr common.Address, hash common.Hash) (*uint256.Int, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return nil, false
	}
	if val, ok := lw.storage[addr][hash]; ok {
		ret := val.(*uint256.Int)
		return ret, true
	}
	return nil, false
}

func (lw *localWrite) getRefund() uint64 {
	return lw.refund
}

func (lw *localWrite) getPrize() *uint256.Int {
	return lw.prize
}

func (lw *localWrite) hasSelfdestructed(addr common.Address) ( /*alive*/ bool /*in_local*/, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return false, false
	}
	if val, ok := lw.storage[addr][utils.EXIST]; ok {
		return !val.(bool), true
	}
	return false, false
}

func (lw *localWrite) get(addr common.Address, hash common.Hash) (interface{}, bool) {
	if _, ok := lw.storage[addr]; !ok {
		return nil, false
	}
	if val, ok := lw.storage[addr][hash]; ok {
		return val, true
	}
	return nil, false
}

// // A functional process

// func (lw *localWrite) exist(addr common.Address) bool {
// 	_, ok := lw.storage[addr]
// 	return ok
// }

// ------------------------- Setters ------------------------------

func (lw *localWrite) setBalance(addr common.Address, balance *uint256.Int) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.BALANCE] = balance
}

func (lw *localWrite) setNonce(addr common.Address, nonce uint64) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.NONCE] = nonce
}

func (lw *localWrite) setCode(addr common.Address, code []byte) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.CODE] = code
}

func (lw *localWrite) setCodeHash(addr common.Address, codeHash common.Hash) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.CODEHASH] = codeHash
}

func (lw *localWrite) setSlot(addr common.Address, hash common.Hash, slot *uint256.Int) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][hash] = slot
}

func (lw *localWrite) setTxContext(thash, bhash common.Hash, txIndex int) {
	lw.thash = thash
	lw.bhash = bhash
	lw.txIndex = txIndex
}

func (lw *localWrite) delete(addr common.Address) {
	lw.setBalance(addr, uint256.NewInt(0))
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.EXIST] = false
}

func (lw *localWrite) addRefund(gas uint64) {
	lw.refund += gas
}

func (lw *localWrite) subRefund(gas uint64) {
	lw.refund -= gas
}

func (lw *localWrite) addLog(log *types.Log) {
	log.TxHash = lw.thash
	log.BlockHash = lw.bhash
	log.TxIndex = uint(lw.txIndex)
	log.Index = lw.logSize
	lw.logs = append(lw.logs, log)
	lw.logSize++
}

func (lw *localWrite) setPrize(amount *uint256.Int) {
	lw.prize = amount
}

func (lw *localWrite) addPrize(amount *uint256.Int) {
	lw.setPrize(new(uint256.Int).Add(lw.prize, amount))
}

// A functional procedure

func (lw *localWrite) createAccount(addr common.Address, _ bool) {
	if _, ok := lw.storage[addr]; !ok {
		lw.storage[addr] = make(map[common.Hash]interface{})
	}
	lw.storage[addr][utils.EXIST] = true
}

func (lw *localWrite) copy() *localWrite {
	c := newLocalWrite()
	c.storage = make(map[common.Address]map[common.Hash]interface{})
	for k, v := range lw.storage {
		c.storage[k] = make(map[common.Hash]interface{})
		for k2, v2 := range v {
			c.storage[k][k2] = v2
		}
	}
	c.refund = lw.refund
	c.thash = lw.thash
	c.bhash = lw.bhash
	c.txIndex = lw.txIndex
	c.logs = make([]*types.Log, len(lw.logs))
	copy(c.logs, lw.logs)
	c.logSize = lw.logSize
	c.prize = new(uint256.Int).Set(lw.prize)
	return c
}
