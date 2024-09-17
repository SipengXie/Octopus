package state

import (
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	types2 "github.com/ledgerwatch/erigon-lib/types"
	"github.com/ledgerwatch/erigon/core/types"
)

type ColdState interface {
	CreateAccount(common.Address, bool)

	SubBalance(common.Address, *uint256.Int)
	AddBalance(common.Address, *uint256.Int)
	GetBalance(common.Address) *uint256.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetCommittedState(common.Address, *common.Hash, *uint256.Int)
	GetState(address common.Address, slot *common.Hash, outValue *uint256.Int)
	SetState(common.Address, *common.Hash, uint256.Int)

	GetTransientState(addr common.Address, key common.Hash) uint256.Int
	SetTransientState(addr common.Address, key common.Hash, value uint256.Int)

	Selfdestruct(common.Address) bool
	HasSelfdestructed(common.Address) bool
	Selfdestruct6780(common.Address)

	Exist(common.Address) bool
	Empty(common.Address) bool

	Prepare(rules *chain.Rules, sender, coinbase common.Address, dest *common.Address,
		precompiles []common.Address, txAccesses types2.AccessList)

	AddressInAccessList(addr common.Address) bool
	AddAddressToAccessList(addr common.Address) (addrMod bool)
	AddSlotToAccessList(addr common.Address, slot common.Hash) (addrMod, slotMod bool)

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPrize(*uint256.Int)
}
