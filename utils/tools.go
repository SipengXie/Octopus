package utils

import (
	"github.com/ledgerwatch/erigon-lib/common"
)

var (
	// DO NOT EXPECT CONFLICTS
	CODE     = common.BytesToHash([]byte("code"))
	CODEHASH = common.BytesToHash([]byte("codeHash"))
	BALANCE  = common.BytesToHash([]byte("balance"))
	NONCE    = common.BytesToHash([]byte("nonce"))
	EXIST    = common.BytesToHash([]byte("exist"))
)

func MakeKey(addr common.Address, hash common.Hash) string {
	return string(addr.Bytes()) + string(hash.Bytes())
}

func ParseKey(key string) (common.Address, common.Hash) {
	addr := common.BytesToAddress([]byte(key[:20]))
	hash := common.BytesToHash([]byte(key[20:]))
	return addr, hash
}

func DecodeHash(hash common.Hash) string {
	switch hash {
	case BALANCE:
		return "balance"
	case NONCE:
		return "nonce"
	case CODEHASH:
		return "codeHash"
	case CODE:
		return "code"
	case EXIST:
		return "exist"
	default:
		return hash.Hex()
	}
}
