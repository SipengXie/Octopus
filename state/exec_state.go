package state

import (
	"blockConcur/rwset"
	"blockConcur/types"
	"blockConcur/utils"
	"bytes"
	"fmt"

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
	SetPrizeKey(coinbase common.Address)
	SetTask(task *types.Task)
}

type ExecState struct {
	// A shared state for the paralle execution of a block
	// Can be concurrently read by multiple goroutines
	// but should not be written to concurrently
	ColdData    ColdState
	LocalWriter *localWrite
	lwSnapshot  *localWrite

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
		ColdData:    ibs,
		LocalWriter: newLocalWrite(),
		lwSnapshot:  nil,
		NewRwSet:    nil,
		OldRwSet:    nil,
		accessList:  newAccessList(),
		Coinbase:    coinbase,
		early_abort: early_abort,
		can_commit:  true,
	}
}

func NewForRun(mvCache *MvCache, coinbase common.Address, early_abort bool) *ExecState {
	coldData := NewExecColdState(mvCache)
	coldData.SetPrizeKey(coinbase)
	return &ExecState{
		ColdData:    coldData,
		LocalWriter: newLocalWrite(),
		lwSnapshot:  nil,
		NewRwSet:    nil,
		OldRwSet:    nil,
		accessList:  newAccessList(),
		Coinbase:    coinbase,
		early_abort: early_abort,
		can_commit:  true,
	}
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
			panic(fmt.Sprintf("invalid read: %s %s", addr.Hex(), utils.DecodeHash(slot)))
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
			panic(fmt.Sprintf("invalid write: %s %s", addr.Hex(), utils.DecodeHash(slot)))
		}
	}
}

func (s *ExecState) SetCoinbase(coinbase common.Address) {
	s.Coinbase = coinbase
	s.ColdData.SetPrizeKey(coinbase)
}

func (s *ExecState) SetTxContext(task *types.Task, newRwSet *rwset.RwSet) {
	s.lwSnapshot = nil
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
	s.LocalWriter.createAccount(addr, contract_created)
	s.NewRwSet.AddWriteSet(addr, utils.EXIST)
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
	s.LocalWriter.setBalance(addr, amount)
}

func (s *ExecState) GetBalance(addr common.Address) *uint256.Int {
	s.is_valid_read(addr, utils.BALANCE)
	s.NewRwSet.AddReadSet(addr, utils.BALANCE)
	balance, ok := s.LocalWriter.getBalance(addr)
	if !ok {
		balance = s.ColdData.GetBalance(addr)
		// if addr == coinbase, we need to add the prize and set the localWrite balance
		if addr == s.Coinbase {
			s.NewRwSet.AddReadSet(addr, utils.PRIZE)
			prize := s.ColdData.GetPrize(s.globalIdx)
			ret := new(uint256.Int).Add(balance, prize)
			s.LocalWriter.setBalance(addr, ret)
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
	s.NewRwSet.AddWriteSet(addr, utils.CODE)
	s.NewRwSet.AddWriteSet(addr, utils.CODEHASH)
	s.LocalWriter.setCode(addr, code)
	s.LocalWriter.setCodeHash(addr, crypto.Keccak256Hash(code))
}

func (s *ExecState) GetCodeSize(addr common.Address) int {
	return len(s.GetCode(addr))
}

func (s *ExecState) AddRefund(gas uint64) {
	s.LocalWriter.addRefund(gas)
	// skip rwset, because the refund is not part of the state
}

func (s *ExecState) SubRefund(gas uint64) {
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
	if !s.Exist(addr) {
		return false
	}
	s.NewRwSet.AddWriteSet(addr, utils.EXIST)
	s.NewRwSet.AddWriteSet(addr, utils.BALANCE)
	s.LocalWriter.delete(addr)
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
	return addrMod
}

func (s *ExecState) AddSlotToAccessList(addr common.Address, slot common.Hash) (addrMod, slotMod bool) {
	addrMod, slotMod = s.accessList.AddSlot(addr, slot)
	return addrMod, slotMod
}

func (s *ExecState) RevertToSnapshot(snapshot int) {
	s.LocalWriter = s.lwSnapshot
	s.lwSnapshot = nil
}

func (s *ExecState) Snapshot() int {
	s.lwSnapshot = s.LocalWriter.copy()
	return 0
}

func (s *ExecState) AddLog(log *types2.Log) {
	s.LocalWriter.addLog(log)
	// skip rwset, because logs are not part of the state
}

// only be called once in a transaction
func (s *ExecState) AddPrize(prize *uint256.Int) {
	s.LocalWriter.addPrize(prize)
	s.NewRwSet.AddWriteSet(s.Coinbase, utils.PRIZE)
}

func (s *ExecState) GetPrize() *uint256.Int {
	return s.LocalWriter.getPrize()
}

// This function is called after the transaction is executed
func (s *ExecState) Commit() {
	if s.can_commit {
		s.ColdData.Commit(s.LocalWriter, s.Coinbase, s.globalIdx)
	} else {
		fmt.Println("CannotCommit", s.globalIdx)
		s.ColdData.Abort()
	}
}
