package types

import (
	"blockConcur/multiversion"

	"github.com/ledgerwatch/erigon-lib/common"
	erigonTypes "github.com/ledgerwatch/erigon/core/types"
)

type Task struct {
	Tid  *multiversion.GlobalId
	Cost uint64
	Msg  erigonTypes.Message

	BlockHash common.Hash
	TxHash    common.Hash

	ReadVersions  map[common.Address]map[common.Hash]*multiversion.Version
	WriteVersions map[common.Address]map[common.Hash]*multiversion.Version
}

func NewTask(id *multiversion.GlobalId, cost uint64, msg erigonTypes.Message, bHash, tHash common.Hash) *Task {
	return &Task{
		Tid:           id,
		Cost:          cost,
		Msg:           msg,
		BlockHash:     bHash,
		TxHash:        tHash,
		ReadVersions:  make(map[common.Address]map[common.Hash]*multiversion.Version),
		WriteVersions: make(map[common.Address]map[common.Hash]*multiversion.Version),
	}
}

func (t *Task) AddReadVersion(addr common.Address, hash common.Hash, version *multiversion.Version) {
	if _, ok := t.ReadVersions[addr]; !ok {
		t.ReadVersions[addr] = make(map[common.Hash]*multiversion.Version)
	}
	t.ReadVersions[addr][hash] = version
}

func (t *Task) AddWriteVersion(addr common.Address, hash common.Hash, version *multiversion.Version) {
	if _, ok := t.WriteVersions[addr]; !ok {
		t.WriteVersions[addr] = make(map[common.Hash]*multiversion.Version)
	}
	t.WriteVersions[addr][hash] = version
}

func (t *Task) Wait() {
	for _, versions := range t.ReadVersions {
		for _, version := range versions {
			version.Mu.Lock()

			for version.Status == multiversion.Pending {
				version.Cond.Wait()
			}

			version.Mu.Unlock()
		}
	}
}
