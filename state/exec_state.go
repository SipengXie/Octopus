package state

import (
	"blockConcur/rwset"
	"blockConcur/types"
	"blockConcur/utils"
	"bytes"
	"fmt"
	"sort"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	types3 "github.com/ledgerwatch/erigon-lib/types"
	types2 "github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/crypto"
)

type ColdState interface {
	Abort()
	Commit(lw *localWrite, coinbase common.Address, TxIdx *utils.ID)
	Empty(addr common.Address) bool
	Exist(addr common.Address) bool
	GetBalance(addr common.Address) *uint256.Int
	GetCode(addr common.Address) []byte
	GetCodeHash(addr common.Address) common.Hash
	GetCodeSize(addr common.Address) int
	GetState(addr common.Address, key *common.Hash, value *uint256.Int)
	GetNonce(addr common.Address) uint64
	GetPrize(TxIdx *utils.ID) *uint256.Int
	HasSelfdestructed(addr common.Address) bool
	SetCoinbase(coinbase common.Address)
	SetTask(task *types.Task)
}

type ExecState struct {
	// A shared state for the paralle execution of a block
	// Can be concurrently read by multiple goroutines
	// but should not be written to concurrently
	ColdData       ColdState
	LocalWriter    *localWrite
	journal        *journal_exec
	validRevisions []revision
	nextRevisionID int

	NewRwSet *rwset.RwSet
	OldRwSet *rwset.RwSet

	Coinbase  common.Address
	globalIdx *utils.ID

	// Per-transaction access list
	// to calculate gas cost
	accessList *accessList
	// if early_abort is true, we will abort the transaction when encountering an invalid read or write
	// outside of the execution, we will use a recover to handle the panic
	early_abort bool
	can_commit  bool
}

func NewForRwSetGen(ibs *IntraBlockState, coinbase common.Address, early_abort bool, cacheSize int) *ExecState {
	return &ExecState{
		ColdData:       ibs,
		LocalWriter:    newLocalWrite(),
		journal:        newJournal_exec(),
		validRevisions: make([]revision, 0),
		nextRevisionID: 0,
		NewRwSet:       nil,
		OldRwSet:       nil,
		accessList:     newAccessList(),
		Coinbase:       coinbase,
		early_abort:    early_abort,
		can_commit:     true,
	}
}

func NewForRun(mvCache *MvCache, coinbase common.Address, early_abort bool) *ExecState {
	coldData := NewExecColdState(mvCache)
	coldData.SetCoinbase(coinbase)
	return &ExecState{
		ColdData:       coldData,
		LocalWriter:    newLocalWrite(),
		journal:        newJournal_exec(),
		validRevisions: make([]revision, 0),
		nextRevisionID: 0,
		NewRwSet:       nil,
		OldRwSet:       nil,
		accessList:     newAccessList(),
		Coinbase:       coinbase,
		early_abort:    early_abort,
		can_commit:     true,
	}
}

type InvalidError struct {
	msg string
}

func (e *InvalidError) Error() string {
	return e.msg
}

func newInvalidError(text string) *InvalidError {
	return &InvalidError{msg: text}
}

// if oldRwSet is nil, we will not check the read set
func (s *ExecState) is_valid_read(addr common.Address, slot common.Hash) {
	if s.OldRwSet == nil {
		return
	}
	ok := s.OldRwSet.ReadSet.Contains(addr, slot)
	if !ok {
		s.can_commit = false
		if s.early_abort {
			panic(newInvalidError(fmt.Sprintf("invalid read: %s %s", addr.Hex(), utils.DecodeHash(slot))))
		}
	}
}

// if oldRwSet is nil, we will not check the write set
func (s *ExecState) is_valid_write(addr common.Address, slot common.Hash) {
	if s.OldRwSet == nil {
		return
	}
	ok := s.OldRwSet.WriteSet.Contains(addr, slot)
	if !ok {
		s.can_commit = false
		if s.early_abort {
			panic(newInvalidError(fmt.Sprintf("invalid write: %s %s", addr.Hex(), utils.DecodeHash(slot))))
		}
	}
}

func (s *ExecState) SetCoinbase(coinbase common.Address) {
	s.Coinbase = coinbase
	s.ColdData.SetCoinbase(coinbase)
}

func (s *ExecState) SetTxContext(task *types.Task, newRwSet *rwset.RwSet) {
	s.journal = newJournal_exec()
	s.validRevisions = make([]revision, 0)
	s.nextRevisionID = 0
	s.LocalWriter = newLocalWrite()
	s.LocalWriter.setTxContext(task.TxHash, task.BlockHash, task.Tid.TxIndex)
	s.globalIdx = task.Tid
	s.OldRwSet = task.RwSet
	s.NewRwSet = newRwSet
	s.can_commit = true
	s.ColdData.SetTask(task)
}

func (s *ExecState) CreateAccount(addr common.Address, contract_created bool) {
	s.is_valid_write(addr, utils.EXIST)
	s.NewRwSet.AddWriteSet(addr, utils.EXIST)
	s.LocalWriter.createAccount(addr, contract_created)
	s.journal.append(createObjectChange{
		account: &addr,
	})
}

func (s *ExecState) SubBalance(addr common.Address, amount *uint256.Int) {
	balance := s.GetBalance(addr)
	newBalance := new(uint256.Int).Set(balance)
	newBalance.Sub(newBalance, amount)
	s.SetBalance(addr, newBalance)
}

func (s *ExecState) AddBalance(addr common.Address, amount *uint256.Int) {
	balance := s.GetBalance(addr)
	newBalance := new(uint256.Int).Set(balance)
	newBalance.Add(newBalance, amount)
	s.SetBalance(addr, newBalance)
}

func (s *ExecState) SetBalance(addr common.Address, amount *uint256.Int) {
	s.is_valid_write(addr, utils.BALANCE)
	s.NewRwSet.AddWriteSet(addr, utils.BALANCE)
	prev, ok := s.LocalWriter.getBalance(addr)
	if !ok {
		prev = uint256.NewInt(0)
	}
	s.LocalWriter.setBalance(addr, amount)
	s.journal.append(balanceChange{
		account: &addr,
		prev:    *prev,
		found:   ok,
	})
}

func (s *ExecState) GetBalance(addr common.Address) *uint256.Int {
	s.is_valid_read(addr, utils.BALANCE)
	s.NewRwSet.AddReadSet(addr, utils.BALANCE)
	balance, ok := s.LocalWriter.getBalance(addr)
	if !ok {
		balance = s.ColdData.GetBalance(addr)
		// if addr == coinbase, we need to add the prize and set the localWrite balance
		if addr == s.Coinbase {
			s.NewRwSet.AddReadPrize()
			prize := s.ColdData.GetPrize(s.globalIdx)
			ret := new(uint256.Int).Add(balance, prize)
			s.LocalWriter.setBalance(addr, ret)
			s.journal.append(balanceChange{
				account: &addr,
				prev:    *balance,
				found:   false,
			})
			return ret
		}
	}
	return balance
}

func (s *ExecState) GetNonce(addr common.Address) uint64 {
	s.is_valid_read(addr, utils.NONCE)
	nonce, ok := s.LocalWriter.getNonce(addr)
	if !ok {
		nonce = s.ColdData.GetNonce(addr)
	}
	s.NewRwSet.AddReadSet(addr, utils.NONCE)
	return nonce
}

func (s *ExecState) SetNonce(addr common.Address, nonce uint64) {
	s.is_valid_write(addr, utils.NONCE)
	prevN, ok := s.LocalWriter.getNonce(addr)
	s.journal.append(nonceChange{
		account: &addr,
		prev:    prevN,
		found:   ok,
	})
	s.LocalWriter.setNonce(addr, nonce)
	s.NewRwSet.AddWriteSet(addr, utils.NONCE)
}

func (s *ExecState) GetCodeHash(addr common.Address) common.Hash {
	s.is_valid_read(addr, utils.CODEHASH)
	s.NewRwSet.AddReadSet(addr, utils.CODEHASH)
	codeHash, ok := s.LocalWriter.getCodeHash(addr)
	if !ok {
		codeHash = s.ColdData.GetCodeHash(addr)
	}
	return codeHash
}

func (s *ExecState) GetCode(addr common.Address) []byte {
	s.is_valid_read(addr, utils.CODE)
	s.NewRwSet.AddReadSet(addr, utils.CODEHASH)
	s.NewRwSet.AddReadSet(addr, utils.CODE)
	code, ok := s.LocalWriter.getCode(addr)
	if !ok {
		code = s.ColdData.GetCode(addr)
	}
	return code
}

func (s *ExecState) SetCode(addr common.Address, code []byte) {
	s.is_valid_write(addr, utils.CODE)
	s.is_valid_write(addr, utils.CODEHASH)
	if dead, found := s.LocalWriter.hasSelfdestructed(addr); dead && found {
		return
	}
	s.NewRwSet.AddWriteSet(addr, utils.CODE)
	s.NewRwSet.AddWriteSet(addr, utils.CODEHASH)
	prevHash, ok1 := s.LocalWriter.getCodeHash(addr)
	prevCode, ok2 := s.LocalWriter.getCode(addr)
	if ok1 != ok2 {
		panic("code hash and code not found at the same time")
	}
	s.journal.append(codeChange{
		account:  &addr,
		prevcode: prevCode,
		prevhash: prevHash,
		found:    ok1,
	})
	s.LocalWriter.setCode(addr, code)
	s.LocalWriter.setCodeHash(addr, crypto.Keccak256Hash(code))
}

func (s *ExecState) GetCodeSize(addr common.Address) int {
	return len(s.GetCode(addr))
}

func (s *ExecState) AddRefund(gas uint64) {
	s.journal.append(refundChange{
		prev: s.LocalWriter.getRefund(),
	})
	s.LocalWriter.addRefund(gas)
	// skip rwset, because the refund is not part of the state
}

func (s *ExecState) SubRefund(gas uint64) {
	s.journal.append(refundChange{
		prev: s.LocalWriter.getRefund(),
	})
	s.LocalWriter.subRefund(gas)
	// skip rwset, because the refund is not part of the state
}

func (s *ExecState) GetRefund() uint64 {
	return s.LocalWriter.getRefund()
	// skip rwset, because the refund is not part of the state
}

// committed state -> read from other transactions
func (s *ExecState) GetCommittedState(addr common.Address, slot *common.Hash, outValue *uint256.Int) {
	s.is_valid_read(addr, *slot)
	s.NewRwSet.AddReadSet(addr, *slot)
	s.ColdData.GetState(addr, slot, outValue)
}

func (s *ExecState) GetState(addr common.Address, slot *common.Hash, outValue *uint256.Int) {
	s.is_valid_read(addr, *slot)
	s.NewRwSet.AddReadSet(addr, *slot)
	v, ok := s.LocalWriter.getSlot(addr, *slot)
	if !ok {
		s.GetCommittedState(addr, slot, outValue)
		return
	}
	*outValue = *v
}

func (s *ExecState) SetState(addr common.Address, slot *common.Hash, value uint256.Int) {
	s.is_valid_write(addr, *slot)
	s.NewRwSet.AddWriteSet(addr, *slot)
	prev, ok := s.LocalWriter.getSlot(addr, *slot)
	if !ok {
		prev = uint256.NewInt(0)
	}
	s.journal.append(storageChange{
		account:  &addr,
		key:      *slot,
		prevalue: *prev,
		found:    ok,
	})

	s.LocalWriter.setSlot(addr, *slot, &value)
}

func (s *ExecState) GetTransientState(addr common.Address, key common.Hash) uint256.Int {
	// Stub implementation
	return uint256.Int{}
}

func (s *ExecState) SetTransientState(addr common.Address, key common.Hash, value uint256.Int) {
	// Stub implementation
}

func (s *ExecState) Selfdestruct(addr common.Address) bool {
	s.is_valid_write(addr, utils.EXIST)
	s.is_valid_write(addr, utils.BALANCE)
	s.is_valid_write(addr, utils.NONCE)
	s.is_valid_write(addr, utils.CODE)
	s.is_valid_write(addr, utils.CODEHASH)
	if !s.Exist(addr) {
		return false
	}
	s.NewRwSet.AddWriteSet(addr, utils.EXIST)
	s.NewRwSet.AddWriteSet(addr, utils.BALANCE)
	s.NewRwSet.AddWriteSet(addr, utils.NONCE)
	s.NewRwSet.AddWriteSet(addr, utils.CODE)
	s.NewRwSet.AddWriteSet(addr, utils.CODEHASH)
	prev, ok1 := s.LocalWriter.hasSelfdestructed(addr)
	prevBalance, ok2 := s.LocalWriter.getBalance(addr)
	if !ok2 {
		prevBalance = uint256.NewInt(0)
	}
	prevNonce, ok3 := s.LocalWriter.getNonce(addr)
	prevCode, ok4 := s.LocalWriter.getCode(addr)
	prevHash, ok5 := s.LocalWriter.getCodeHash(addr)

	s.journal.append(selfdestructChange{
		account:        &addr,
		prev:           !prev,
		prevbalance:    *prevBalance,
		prevnonce:      prevNonce,
		prevcode:       prevCode,
		prevhash:       prevHash,
		found_exist:    ok1,
		found_balance:  ok2,
		found_nonce:    ok3,
		found_code:     ok4,
		found_codehash: ok5,
	})
	s.LocalWriter.delete(addr)
	s.LocalWriter.setBalance(addr, uint256.NewInt(0))
	s.LocalWriter.setNonce(addr, 0)
	s.LocalWriter.setCode(addr, nil)
	s.LocalWriter.setCodeHash(addr, common.Hash{})
	return true
}

func (s *ExecState) HasSelfdestructed(addr common.Address) bool {
	s.is_valid_read(addr, utils.EXIST)
	s.NewRwSet.AddReadSet(addr, utils.EXIST)
	selfdestructed, in_local := s.LocalWriter.hasSelfdestructed(addr)
	if !in_local {
		selfdestructed = s.ColdData.HasSelfdestructed(addr)
	}
	return selfdestructed
}

func (s *ExecState) Selfdestruct6780(addr common.Address) {
	s.Selfdestruct(addr)
}

func (s *ExecState) Exist(addr common.Address) bool {
	s.is_valid_read(addr, utils.EXIST)
	s.NewRwSet.AddReadSet(addr, utils.EXIST)
	_, ok := s.LocalWriter.storage[addr]
	if !ok {
		ok = s.ColdData.Exist(addr)
	}
	return ok
}

func (s *ExecState) Empty(addr common.Address) bool {
	s.is_valid_read(addr, utils.BALANCE)
	s.is_valid_read(addr, utils.NONCE)
	s.is_valid_read(addr, utils.CODEHASH)
	s.NewRwSet.AddReadSet(addr, utils.BALANCE)
	s.NewRwSet.AddReadSet(addr, utils.NONCE)
	s.NewRwSet.AddReadSet(addr, utils.CODEHASH)
	exist := s.Exist(addr)
	if !exist {
		return true
	}
	balance := s.GetBalance(addr)
	nonce := s.GetNonce(addr)
	codeHash := s.GetCodeHash(addr)

	return balance.IsZero() && nonce == 0 && bytes.Equal(codeHash[:], emptyCodeHash)
}

// things about access list are costy and not used in the current implementation
// may ignore them
func (s *ExecState) Prepare(rules *chain.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list types3.AccessList) {
	if rules.IsBerlin {
		// Clear out any leftover from previous executions
		al := newAccessList()
		s.accessList = al

		al.AddAddress(sender)
		if dst != nil {
			al.AddAddress(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range precompiles {
			al.AddAddress(addr)
		}
		for _, el := range list {
			al.AddAddress(el.Address)
			for _, key := range el.StorageKeys {
				al.AddSlot(el.Address, key)
			}
		}
		if rules.IsShanghai { // EIP-3651: warm coinbase
			al.AddAddress(coinbase)
		}
	}
	// Reset transient storage at the beginning of transaction execution
	// s.transientStorage = newTransientStorage()
}

func (s *ExecState) AddressInAccessList(addr common.Address) bool {
	return s.accessList.ContainsAddress(addr)
}

func (s *ExecState) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return s.accessList.Contains(addr, slot)
}

func (s *ExecState) AddAddressToAccessList(addr common.Address) (addrMod bool) {
	addrMod = s.accessList.AddAddress(addr)
	if addrMod {
		s.journal.append(accessListAddAccountChange{&addr})
	}
	return addrMod
}

func (s *ExecState) AddSlotToAccessList(addr common.Address, slot common.Hash) (addrMod, slotMod bool) {
	addrMod, slotMod = s.accessList.AddSlot(addr, slot)
	if addrMod {
		// In practice, this should not happen, since there is no way to enter the
		// scope of 'address' without having the 'address' become already added
		// to the access list (via call-variant, create, etc).
		// Better safe than sorry, though
		s.journal.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		s.journal.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
	return addrMod, slotMod
}

func (s *ExecState) RevertToSnapshot(snapshot int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= snapshot
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != snapshot {
		panic(fmt.Errorf("revision id %v cannot be reverted", snapshot))
	}
	temp := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.journal.revert(s, temp)
	s.validRevisions = s.validRevisions[:idx]
}

func (s *ExecState) Snapshot() int {
	id := s.nextRevisionID
	s.nextRevisionID++
	s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
	return id
}

func (s *ExecState) AddLog(log *types2.Log) {
	s.LocalWriter.addLog(log)
	// skip rwset, because logs are not part of the state
}

// only be called once in a transaction
func (s *ExecState) AddPrize(prize *uint256.Int) {
	s.LocalWriter.addPrize(prize)
	s.NewRwSet.AddWritePrize()
}

func (s *ExecState) GetPrize() *uint256.Int {
	return s.LocalWriter.getPrize()
}

// This function is called after the transaction is executed
func (s *ExecState) Commit() {
	if s.can_commit {
		s.ColdData.Commit(s.LocalWriter, s.Coinbase, s.globalIdx)
	} else {
		// fmt.Println("CannotCommit", s.globalIdx)
		s.ColdData.Abort()
		panic(fmt.Errorf("CannotCommit %v", s.globalIdx))
	}
}
