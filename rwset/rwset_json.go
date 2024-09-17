package rwset

import (
	"encoding/json"

	"github.com/ledgerwatch/erigon-lib/common"
)

type RWSetJson struct {
	ReadSet  map[common.Address][]string `json:"readSet"`
	WriteSet map[common.Address][]string `json:"writeSet"`
}

func (rwj RWSetJson) ToString() string {
	b, _ := json.Marshal(rwj)
	return string(b)
}
