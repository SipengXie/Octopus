// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"octopus/utils"
	"sync"

	"github.com/holiman/uint256"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
)

// journalEntry_ibs is a modification entry in the state change journal that can be
// reverted on demand.
type journalEntry_ibs interface {
	// revert undoes the changes introduced by this journal entry.
	revert(*IntraBlockState)

	// dirtied returns the Ethereum address modified by this journal entry.
	dirtied() *libcommon.Address
}

type journalEntry_exec interface {
	// revert undoes the changes introduced by this journal entry.
	revertExec(*ExecState)

	// dirtied returns the Ethereum address modified by this journal entry.
	dirtied() *libcommon.Address
}

// journal_ibs contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in case of an execution
// exception or revertal request.
type journal_ibs struct {
	entries []journalEntry_ibs // Current changes tracked by the journal
	dirties sync.Map           // Dirty accounts and the number of changes
}

// newJournal_ibs create a new initialized journal.
func newJournal_ibs() *journal_ibs {
	return &journal_ibs{
		dirties: sync.Map{},
	}
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal_ibs) append(entry journalEntry_ibs) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		count, _ := j.dirties.LoadOrStore(*addr, 0)
		j.dirties.Store(*addr, count.(int)+1)
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// dirty handling too.
func (j *journal_ibs) revert(statedb *IntraBlockState, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].dirtied(); addr != nil {
			count, _ := j.dirties.Load(*addr)
			if count.(int) == 1 {
				j.dirties.Delete(*addr)
			} else {
				j.dirties.Store(*addr, count.(int)-1)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. This method is an ugly hack to handle the RIPEMD
// precompile consensus exception.
func (j *journal_ibs) dirty(addr libcommon.Address) {
	count, _ := j.dirties.LoadOrStore(addr, 0)
	j.dirties.Store(addr, count.(int)+1)
}

// length returns the current number of entries in the journal.
func (j *journal_ibs) length() int {
	return len(j.entries)
}

type journal_exec struct {
	entries []journalEntry_exec
	dirties map[libcommon.Address]int
}

func newJournal_exec() *journal_exec {
	return &journal_exec{
		entries: make([]journalEntry_exec, 0),
		dirties: make(map[libcommon.Address]int),
	}
}

func (j *journal_exec) append(entry journalEntry_exec) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

func (j *journal_exec) revert(statedb *ExecState, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].revertExec(statedb)

		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr] > 1 {
				j.dirties[*addr]--
			} else {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

func (j *journal_exec) length() int {
	return len(j.entries)
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *libcommon.Address
	}
	resetObjectChange struct {
		account *libcommon.Address
		prev    *stateObject
	}
	selfdestructChange struct {
		account     *libcommon.Address
		prev        bool // whether account had already selfdestructed
		prevbalance uint256.Int
		// prevnonce      uint64
		// prevcode       []byte
		// prevhash       libcommon.Hash
		found_exist   bool
		found_balance bool
		// found_nonce    bool
		// found_code     bool
		// found_codehash bool
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *libcommon.Address
		prev    uint256.Int
		found   bool
	}
	balanceIncrease struct {
		account  *libcommon.Address
		increase uint256.Int
	}
	balanceIncreaseTransfer struct {
		bi *BalanceIncrease
	}
	nonceChange struct {
		account *libcommon.Address
		prev    uint64
		found   bool
	}
	storageChange struct {
		account  *libcommon.Address
		key      libcommon.Hash
		prevalue uint256.Int
		found    bool
	}
	fakeStorageChange struct {
		account  *libcommon.Address
		key      libcommon.Hash
		prevalue uint256.Int
	}
	codeChange struct {
		account  *libcommon.Address
		prevcode []byte
		prevhash libcommon.Hash
		found    bool
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash libcommon.Hash
	}
	touchChange struct {
		account *libcommon.Address
	}

	// Changes to the access list
	accessListAddAccountChange struct {
		address *libcommon.Address
	}
	accessListAddSlotChange struct {
		address *libcommon.Address
		slot    *libcommon.Hash
	}

	transientStorageChange struct {
		account  *libcommon.Address
		key      libcommon.Hash
		prevalue uint256.Int
	}
)

func (ch createObjectChange) revert(s *IntraBlockState) {
	s.stateObjects.Delete(*ch.account)
	s.stateObjectsDirty.Delete(*ch.account)
}

func (ch createObjectChange) revertExec(s *ExecState) {
	delete(s.LocalWriter.storage, *ch.account)
}

func (ch createObjectChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch resetObjectChange) revert(s *IntraBlockState) {
	s.setStateObject(*ch.account, ch.prev)
}

func (ch resetObjectChange) dirtied() *libcommon.Address {
	return nil
}

func (ch selfdestructChange) revert(s *IntraBlockState) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.selfdestructed = ch.prev
		obj.setBalance(&ch.prevbalance)
	}
}

func (ch selfdestructChange) revertExec(s *ExecState) {
	if !ch.found_exist {
		delete(s.LocalWriter.storage[*ch.account], utils.EXIST)
	} else {
		s.LocalWriter.storage[*ch.account][utils.EXIST] = ch.prev
	}
	if !ch.found_balance {
		delete(s.LocalWriter.storage[*ch.account], utils.BALANCE)
	} else {
		s.LocalWriter.storage[*ch.account][utils.BALANCE] = &ch.prevbalance
	}
	// if !ch.found_nonce {
	// 	delete(s.LocalWriter.storage[*ch.account], utils.NONCE)
	// } else {
	// 	s.LocalWriter.storage[*ch.account][utils.NONCE] = ch.prevnonce
	// }
	// if !ch.found_code {
	// 	delete(s.LocalWriter.storage[*ch.account], utils.CODE)
	// } else {
	// 	s.LocalWriter.storage[*ch.account][utils.CODE] = ch.prevcode
	// }
	// if !ch.found_codehash {
	// 	delete(s.LocalWriter.storage[*ch.account], utils.CODEHASH)
	// } else {
	// 	s.LocalWriter.storage[*ch.account][utils.CODEHASH] = ch.prevhash
	// }
	if len(s.LocalWriter.storage[*ch.account]) == 0 {
		delete(s.LocalWriter.storage, *ch.account)
	}
}

func (ch selfdestructChange) dirtied() *libcommon.Address {
	return ch.account
}

var ripemd = libcommon.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) revert(s *IntraBlockState) {
}

func (ch touchChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch balanceChange) revert(s *IntraBlockState) {
	s.getStateObject(*ch.account).setBalance(&ch.prev)
}

func (ch balanceChange) revertExec(s *ExecState) {
	if !ch.found {
		delete(s.LocalWriter.storage[*ch.account], utils.BALANCE)
		if len(s.LocalWriter.storage[*ch.account]) == 0 {
			delete(s.LocalWriter.storage, *ch.account)
		}
	} else {
		s.LocalWriter.storage[*ch.account][utils.BALANCE] = &ch.prev
	}
}

func (ch balanceChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch balanceIncrease) revert(s *IntraBlockState) {
	if bi, ok := s.balanceInc[*ch.account]; ok {
		bi.increase.Sub(&bi.increase, &ch.increase)
		bi.count--
		if bi.count == 0 {
			delete(s.balanceInc, *ch.account)
		}
	}
}

func (ch balanceIncrease) dirtied() *libcommon.Address {
	return ch.account
}

func (ch balanceIncreaseTransfer) dirtied() *libcommon.Address {
	return nil
}

func (ch balanceIncreaseTransfer) revert(s *IntraBlockState) {
	ch.bi.transferred = false
}
func (ch nonceChange) revert(s *IntraBlockState) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch nonceChange) revertExec(s *ExecState) {
	if !ch.found {
		delete(s.LocalWriter.storage[*ch.account], utils.NONCE)
		if len(s.LocalWriter.storage[*ch.account]) == 0 {
			delete(s.LocalWriter.storage, *ch.account)
		}
	} else {
		s.LocalWriter.storage[*ch.account][utils.NONCE] = ch.prev
	}
}

func (ch nonceChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch codeChange) revert(s *IntraBlockState) {
	s.getStateObject(*ch.account).setCode(ch.prevhash, ch.prevcode)
}

func (ch codeChange) revertExec(s *ExecState) {
	if !ch.found {
		delete(s.LocalWriter.storage[*ch.account], utils.CODE)
		delete(s.LocalWriter.storage[*ch.account], utils.CODEHASH)
		if len(s.LocalWriter.storage[*ch.account]) == 0 {
			delete(s.LocalWriter.storage, *ch.account)
		}
	} else {
		s.LocalWriter.storage[*ch.account][utils.CODE] = ch.prevcode
		s.LocalWriter.storage[*ch.account][utils.CODEHASH] = ch.prevhash
	}
}

func (ch codeChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch storageChange) revert(s *IntraBlockState) {
	s.getStateObject(*ch.account).setState(&ch.key, ch.prevalue)
}

func (ch storageChange) revertExec(s *ExecState) {
	if !ch.found {
		delete(s.LocalWriter.storage[*ch.account], ch.key)
		if len(s.LocalWriter.storage[*ch.account]) == 0 {
			delete(s.LocalWriter.storage, *ch.account)
		}
	} else {
		s.LocalWriter.storage[*ch.account][ch.key] = &ch.prevalue
	}
}

func (ch storageChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch fakeStorageChange) revert(s *IntraBlockState) {
	s.getStateObject(*ch.account).fakeStorage.Set(ch.key, ch.prevalue)
}

func (ch fakeStorageChange) dirtied() *libcommon.Address {
	return ch.account
}

func (ch transientStorageChange) revert(s *IntraBlockState) {
	s.setTransientState(*ch.account, ch.key, ch.prevalue)
}

func (ch transientStorageChange) dirtied() *libcommon.Address {
	return nil
}

func (ch refundChange) revert(s *IntraBlockState) {
	s.refund = ch.prev
}

func (ch refundChange) revertExec(s *ExecState) {
	s.LocalWriter.refund = ch.prev
}

func (ch refundChange) dirtied() *libcommon.Address {
	return nil
}

func (ch addLogChange) revert(s *IntraBlockState) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}

func (ch addLogChange) dirtied() *libcommon.Address {
	return nil
}

func (ch accessListAddAccountChange) revert(s *IntraBlockState) {
	/*
		One important invariant here, is that whenever a (addr, slot) is added, if the
		addr is not already present, the add causes two journal entries:
		- one for the address,
		- one for the (address,slot)
		Therefore, when unrolling the change, we can always blindly delete the
		(addr) at this point, since no storage adds can remain when come upon
		a single (addr) change.
	*/
	s.accessList.DeleteAddress(*ch.address)
}

func (ch accessListAddAccountChange) revertExec(s *ExecState) {
	s.accessList.DeleteAddress(*ch.address)
}

func (ch accessListAddAccountChange) dirtied() *libcommon.Address {
	return nil
}

func (ch accessListAddSlotChange) revert(s *IntraBlockState) {
	s.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) revertExec(s *ExecState) {
	s.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) dirtied() *libcommon.Address {
	return nil
}
