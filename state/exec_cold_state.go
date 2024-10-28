package state

import (
	mv "blockConcur/multiversion"
	"blockConcur/types"
	"blockConcur/utils"
	"bytes"
	"fmt"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/crypto"
)

var emptyCodeHash = crypto.Keccak256Hash(nil)

func isEmptyCodeHash(codeHash common.Hash) bool {
	if codeHash == (common.Hash{}) {
		return true
	}
	return bytes.Equal(codeHash[:], emptyCodeHash[:])
}

type versionMap struct {
	data map[string]*mv.Version
}

func newVersionMap(data map[string]*mv.Version) *versionMap {
	return &versionMap{
		data: data,
	}
}

func (vm *versionMap) get(addr common.Address, hash common.Hash) *mv.Version {
	key := utils.MakeKey(addr, hash)
	return vm.data[key]
}

type ExecColdState struct {
	// a shared uint256.Int
	input_predict  *versionMap   // some pointers of the inner_state
	output_predict *versionMap   // some pointers of the inner_state, only used in commit_localwrite
	prize_predict  []*mv.Version // some pointers of the inner_state, only used in commit_localwrite
	inner_state    *MvCache      // the same level as the exec_cold_states, for data that are not in input and output
}

func NewExecColdState(mvc *MvCache) *ExecColdState {
	return &ExecColdState{
		inner_state: mvc,
	}
}

func (s *ExecColdState) SetTask(task *types.Task) {
	s.input_predict = newVersionMap(task.ReadVersions)
	s.output_predict = newVersionMap(task.WriteVersions)
	s.prize_predict = task.PrizeVersions
}

func (s *ExecColdState) SetCoinbase(coinbase common.Address) {
	s.inner_state.SetCoinbase(coinbase)
}

func (s *ExecColdState) GetBalance(addr common.Address) *uint256.Int {
	var balance *uint256.Int
	var ok bool
	version := s.input_predict.get(addr, utils.BALANCE).GetVisible()
	if version == nil {
		balance, ok = s.inner_state.Fetch(addr, utils.BALANCE).(*uint256.Int)
		if !ok {
			panic("balance is not a uint256.Int")
		}
		return balance
	}
	balance, ok = version.Data.(*uint256.Int)
	if !ok {
		temp := s.input_predict.get(addr, utils.BALANCE)
		fmt.Println(temp)
		panic("balance is not a uint256.Int")
	}
	return balance
}

func (s *ExecColdState) GetNonce(addr common.Address) uint64 {
	var nonce uint64
	var ok bool
	version := s.input_predict.get(addr, utils.NONCE).GetVisible()
	if version == nil {
		nonce, ok = s.inner_state.Fetch(addr, utils.NONCE).(uint64)
		if !ok {
			panic("nonce is not a uint64")
		}
		return nonce
	}
	nonce, ok = version.Data.(uint64)
	if !ok {
		temp := s.input_predict.get(addr, utils.NONCE)
		fmt.Println(temp)
		panic("nonce is not a uint64")
	}
	return nonce
}

func (s *ExecColdState) GetCodeHash(addr common.Address) common.Hash {
	var codeHash common.Hash
	var ok bool
	version := s.input_predict.get(addr, utils.CODEHASH).GetVisible()
	if version == nil {
		codeHash, ok = s.inner_state.Fetch(addr, utils.CODEHASH).(common.Hash)
		if !ok {
			panic("codeHash is not a common.Hash")
		}
		return codeHash
	}
	codeHash, ok = version.Data.(common.Hash)
	if !ok {
		temp := s.input_predict.get(addr, utils.CODEHASH)
		fmt.Println(temp)
		panic("codeHash is not a common.Hash")
	}
	return codeHash
}

func (s *ExecColdState) GetCode(addr common.Address) []byte {
	var code []byte
	var ok bool
	version := s.input_predict.get(addr, utils.CODE).GetVisible()
	if version == nil {
		code, ok = s.inner_state.Fetch(addr, utils.CODE).([]byte)
		if !ok {
			panic("code is not a []byte")
		}
		return code
	}
	code, ok = version.Data.([]byte)
	if !ok {
		temp := s.input_predict.get(addr, utils.CODE)
		fmt.Println(temp)
		panic("code is not a []byte")
	}
	return code
}

func (s *ExecColdState) GetCodeSize(addr common.Address) int {
	return (len(s.GetCode(addr)))
}

func (s *ExecColdState) GetState(addr common.Address, hash *common.Hash, value *uint256.Int) {
	version := s.input_predict.get(addr, *hash).GetVisible()
	if version == nil {
		slot, ok := s.inner_state.Fetch(addr, *hash).(*uint256.Int)
		if !ok {
			panic("value is not a *uint256.Int")
		}
		value.Set(slot)
		return
	}
	slot, ok := version.Data.(*uint256.Int)
	if !ok {
		temp := s.input_predict.get(addr, *hash)
		fmt.Println(temp)
		panic("slot is not a *uint256.Int")
	}
	value.Set(slot)
}

func (s *ExecColdState) Exist(addr common.Address) bool {
	var exist, ok bool
	version := s.input_predict.get(addr, utils.EXIST).GetVisible()
	if version == nil {
		exist, ok = s.inner_state.Fetch(addr, utils.EXIST).(bool)
		if !ok {
			panic("exist is not a bool")
		}
		return exist
	}
	exist, ok = version.Data.(bool)
	if !ok {
		temp := s.input_predict.get(addr, utils.EXIST)
		fmt.Println(temp)
		panic("exist is not a bool")
	}
	return exist
}

func (s *ExecColdState) Empty(addr common.Address) bool {
	balance := s.GetBalance(addr)
	nonce := s.GetNonce(addr)
	codeHash := s.GetCodeHash(addr)
	return balance.IsZero() && nonce == 0 && isEmptyCodeHash(codeHash)
}

func (s *ExecColdState) HasSelfdestructed(addr common.Address) bool {
	return !s.Exist(addr)
}

func (s *ExecColdState) GetPrize(TxIdx *utils.ID) *uint256.Int {
	if len(s.prize_predict) == 0 {
		return s.inner_state.FetchPrize(TxIdx)
	}
	ret := uint256.NewInt(0)
	for _, version := range s.prize_predict {
		version.Wait()
		if version.Status == mv.Committed {
			ret.Add(ret, version.Data.(*uint256.Int))
		}
	}
	return ret
	// return s.inner_state.FetchPrize(TxIdx)
}

// if we entered Commit function, then the localwrite will merge to
// the output_predict, and update the inner_state utilize the new output_predict
func (s *ExecColdState) Commit(lw *localWrite, coinbase common.Address, TxIdx *utils.ID) {
	if len(s.output_predict.data) == 0 {
		s.commitWithoutOutput(lw, coinbase, TxIdx)
		return
	}
	prize := lw.getPrize()
	pVersion := s.output_predict.data["prize"]
	for key, version := range s.output_predict.data {
		if key == "prize" {
			continue
		}
		addr, hash := utils.ParseKey(key)
		// if the addr & hash is not in the lw, settle the version to ignore
		value, ok := lw.get(addr, hash)
		if !ok {
			version.Settle(mv.Ignore, nil)
		} else {
			s.inner_state.Update(version, key, value)
			if hash == utils.BALANCE && addr == coinbase {
				// clear the prize
				for _, version := range s.prize_predict {
					version.Data = uint256.NewInt(0)
				}
				// s.inner_state.PrunePrize(TxIdx)
			}
		}
	}
	s.inner_state.UpdatePrize(pVersion, prize)
}

func (s *ExecColdState) Abort() {
	for _, version := range s.output_predict.data {
		version.Settle(mv.Ignore, nil)
	}
}

// this function is used for serial execution committment
// we will generate versions for the TxIdx and install them to the version chain
func (s *ExecColdState) commitWithoutOutput(lw *localWrite, coinbase common.Address, TxIdx *utils.ID) {
	prize := lw.getPrize()
	pVersion := mv.NewVersion(prize, TxIdx, mv.Committed)
	s.inner_state.InsertVersion("prize", pVersion)
	s.inner_state.UpdatePrize(pVersion, prize)
	for addr, cache := range lw.storage {
		for hash, value := range cache {
			version := mv.NewVersion(value, TxIdx, mv.Committed)
			s.inner_state.InsertVersion(utils.MakeKey(addr, hash), version)
			s.inner_state.Update(version, utils.MakeKey(addr, hash), value)
			if hash == utils.BALANCE && addr == coinbase {
				s.inner_state.PrunePrize(TxIdx)
			}
		}
	}

}
