package mockenv

import (
	"encoding/binary"
	"octopus/state"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/kv"
	state2 "github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/turbo/rpchelper"
)

func makePreState(rules *chain.Rules, tx kv.RwTx, accounts types.GenesisAlloc, blockNr uint64) (*state.IntraBlockState, error) {
	r := rpchelper.NewLatestStateReader(tx)
	statedb := state.New(r)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		balance := uint256.NewInt(0)
		if a.Balance != nil {
			balance, _ = uint256.FromBig(a.Balance)
		}
		statedb.SetBalance(addr, balance)
		for k, v := range a.Storage {
			key := k
			val := uint256.NewInt(0).SetBytes(v.Bytes())
			statedb.SetState(addr, &key, *val)
		}

		if len(a.Code) > 0 || len(a.Storage) > 0 {
			statedb.SetIncarnation(addr, state2.FirstContractIncarnation)

			var b [8]byte
			binary.BigEndian.PutUint64(b[:], state2.FirstContractIncarnation)
			if err := tx.Put(kv.IncarnationMap, addr[:], b[:]); err != nil {
				return nil, err
			}
		}
	}

	// var w state2.StateWriter
	// if config3.EnableHistoryV4InTest {
	// 	panic("implement me")
	// } else {
	// 	w = state2.NewPlainStateWriter(tx, nil, blockNr+1)
	// }
	// statedb.SetWriter(w)
	// Commit and re-open to start with a clean state.
	// if err := statedb.FinalizeTx(rules, w); err != nil {
	// 	return nil, err
	// }
	// if err := statedb.CommitBlock(rules, w); err != nil {
	// 	return nil, err
	// }
	return statedb, nil
}
