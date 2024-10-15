package mockenv

import (
	"blockConcur/state"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/turbo/stages/mock"
)

type AccountState struct {
	Nonce    *string            `json:"nonce,omitempty"`
	CodeHash *string            `json:"codeHash,omitempty"`
	Code     *string            `json:"code,omitempty"`
	Balance  *string            `json:"balance,omitempty"`
	Storage  *map[string]string `json:"storage,omitempty"`
}

func (a *AccountState) SetDefaults() {
	if a.Nonce == nil {
		nonce := "0"
		a.Nonce = &nonce
	}
	if a.Code == nil {
		code := ""
		a.Code = &code
	}
	if a.Balance == nil {
		balance := "0"
		a.Balance = &balance
	}
	if a.Storage == nil {
		storage := map[string]string{}
		a.Storage = &storage
	}
}

func GetAllocation(block BlockState) (*types.GenesisAlloc, error) {
	alloc := types.GenesisAlloc{}
	for addressHex, accountState := range block.Pre {
		accountState.SetDefaults()
		address := common.HexToAddress(addressHex)
		nonce, err1 := strconv.ParseInt(*accountState.Nonce, 16, 64)
		code, err2 := hex.DecodeString(*accountState.Code)
		balanceBigint := new(big.Int)
		balance, ok3 := balanceBigint.SetString(*accountState.Balance, 16)
		if err1 != nil || err2 != nil || !ok3 {
			return nil, fmt.Errorf("failed to parse data: [1]%v [2]%v [3]%v", err1, err2, ok3)
		}

		storage := map[common.Hash]common.Hash{}
		for key, value := range *accountState.Storage {
			keyBytes, err4 := hex.DecodeString(key[2:]) // With '0x' prefix
			valueBytes, err5 := hex.DecodeString(value) // Without '0x' prefix
			if err4 != nil || err5 != nil {
				return nil, fmt.Errorf("failed to parse storage: [4]%v [5]%v", err4, err5)
			}
			storage[common.BytesToHash(keyBytes)] = common.BytesToHash(valueBytes)
		}

		alloc[address] = types.GenesisAccount{
			Nonce:   uint64(nonce),
			Code:    code,
			Balance: balance,
			Storage: storage,
		}
	}
	return &alloc, nil
}

type BlockState struct {
	BlockNumber uint64                  `json:"blockNumber"`
	Pre         map[string]AccountState `json:"pre"`
	Post        map[string]AccountState `json:"post"`
}

func getBlockStates(startHeight uint64) []BlockState {
	filepath := filepath.Join("/chaindata/statedata", fmt.Sprintf("block%d-%d.json", startHeight, startHeight+999))
	file, err := os.Open(filepath)
	if err != nil {
		panic(fmt.Sprintf("failed to load file: %v", err))
	}
	defer file.Close()
	var blockStates []BlockState
	fmt.Println("getState: decoding json, might take a while...")
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&blockStates); err != nil {
		panic(fmt.Sprintf("failed to decode json: %v", err))
	} else {
		fmt.Println("getState: decode success")
	}
	return blockStates
}

type FetchTransactions struct {
	Transactions types.Transactions `json:"transactions"`
}

func (b *FetchTransactions) UnmarshalJSON(data []byte) error {
	type Alias FetchTransactions
	aux := &struct {
		Transactions []json.RawMessage `json:"transactions"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	for _, rawTx := range aux.Transactions {
		tx, err := types.UnmarshalTransactionFromJSON(rawTx)
		if err != nil {
			return err
		}
		b.Transactions = append(b.Transactions, tx)
	}

	return nil
}

func getTransactions(startHeight uint64) []*FetchTransactions {
	filepath := filepath.Join("/chaindata/blockdata", fmt.Sprintf("block%d-%d.json", startHeight, startHeight+999))
	file, err := os.Open(filepath)
	if err != nil {
		panic(fmt.Sprintf("failed to load file: %v", err))
	}
	defer file.Close()
	var FT []*FetchTransactions
	fmt.Println("getTransactions: decoding json, might take a while...")
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&FT); err != nil {
		panic(fmt.Sprintf("failed to decode json: %v", err))
	} else {
		fmt.Println("getTransactions: decode success")
	}
	return FT
}

type FetchHeader struct {
	Header *types.Header `json:"header"`
}

func getHeaders() []*types.Header {
	startHeight := uint64(18500000)
	endHeight := uint64(18520000)
	headersChan := make(chan *types.Header)
	var wg sync.WaitGroup

	for start := startHeight; start < endHeight; start += 1000 {
		wg.Add(1)
		go func(start uint64) {
			defer wg.Done()
			filepath := filepath.Join("/chaindata/blockdata", fmt.Sprintf("block%d-%d.json", start, start+999))
			file, err := os.Open(filepath)
			if err != nil {
				panic(fmt.Sprintf("failed to load file: %v", err))
			}
			defer file.Close()

			var FH []*FetchHeader
			fmt.Println("getHeaders: decoding json for block", start)
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&FH); err != nil {
				fmt.Printf("failed to decode json for block %d-%d: %v\n", start, start+999, err)
				return
			}

			for _, f := range FH {
				headersChan <- f.Header
			}
		}(start)
	}

	go func() {
		wg.Wait()
		close(headersChan)
	}()

	headers := make([]*types.Header, 0)
	for header := range headersChan {
		headers = append(headers, header)
	}

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Number.Uint64() < headers[j].Number.Uint64()
	})

	return headers
}

// txs from startHeight to startHeight+999.
// headers from 18500000 to 18520000.
// states from startHeight to startHeight+999.
// dbtxs from startHeight to startHeight+999.
func GetMainnetData(t *testing.T, startHeight uint64) ([]*FetchTransactions, []*types.Header, []*state.IntraBlockState, []*state.IntraBlockState, []kv.RwTx) {
	FT := getTransactions(startHeight)
	blockStates := getBlockStates(startHeight)
	headers := getHeaders()
	states := make([]*state.IntraBlockState, len(blockStates))
	states_bak := make([]*state.IntraBlockState, len(blockStates))
	dbtxs := make([]kv.RwTx, len(blockStates))
	fmt.Println("GetMainData: making prestate...")
	for i, blockState := range blockStates {
		height := uint64(i) + startHeight
		alloc, err := GetAllocation(blockState)
		if err != nil {
			panic(fmt.Sprintf("failed to get allocation: %v", err))
		}
		m := mock.Mock(t)
		tx, err := m.DB.BeginRw(m.Ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to begin rw: %v", err))
		}
		rules := params.AllProtocolChanges.Rules(height, headers[height-18500000].Time)
		states[i], _ = makePreState(rules, tx, *alloc, height)
		states_bak[i], _ = makePreState(rules, tx, *alloc, height)
		dbtxs[i] = tx
	}
	fmt.Println("GetMainData: success")
	return FT, headers, states, states_bak, dbtxs
}

// Get a list of state databases for the specified starting height
func GetStateDBs(t *testing.T, startHeight uint64) []*state.IntraBlockState {
	blockStates := getBlockStates(startHeight)
	headers := getHeaders()
	states := make([]*state.IntraBlockState, len(blockStates))

	fmt.Println("Getting state databases: Creating pre-states...")
	for i, blockState := range blockStates {
		height := uint64(i) + startHeight
		alloc, err := GetAllocation(blockState)
		if err != nil {
			panic(fmt.Sprintf("Failed to get allocation: %v", err))
		}
		m := mock.Mock(t)
		tx, err := m.DB.BeginRw(m.Ctx)
		if err != nil {
			panic(fmt.Sprintf("Failed to begin read-write transaction: %v", err))
		}
		defer tx.Rollback()

		rules := params.AllProtocolChanges.Rules(height, headers[height-18500000].Time)
		states[i], _ = makePreState(rules, tx, *alloc, height)
	}
	fmt.Println("Getting state databases: Success")
	return states
}
