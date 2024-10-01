package utils

type ID struct {
	BlockNumber uint64
	Incarnation int // used for re-executed transaction, prioritize TxIndex
	TxIndex     int
}

const MAXUINT64 = ^uint64(0)

var SnapshotID = &ID{BlockNumber: 0, TxIndex: -1, Incarnation: -1}
var EndID = &ID{BlockNumber: MAXUINT64, TxIndex: -1, Incarnation: -1}

func NewID(blockNumber uint64, txIndex int, incarnation int) *ID {
	return &ID{
		BlockNumber: blockNumber,
		TxIndex:     txIndex,
		Incarnation: incarnation,
	}
}

func (g *ID) Equal(other *ID) bool {
	return g.BlockNumber == other.BlockNumber && g.TxIndex == other.TxIndex && g.Incarnation == other.Incarnation
}

func (g *ID) Less(other *ID) bool {
	if g.BlockNumber < other.BlockNumber {
		return true
	}
	if g.BlockNumber == other.BlockNumber {
		if g.Incarnation == other.Incarnation {
			return g.TxIndex < other.TxIndex
		} else {
			return g.Incarnation < other.Incarnation
		}
	}
	return false
}

type IDs []*ID

func (ids IDs) Find(target *ID) (int, bool) {
	left, right := 0, len(ids)-1
	for left <= right {
		mid := (left + right) / 2
		if ids[mid].Equal(target) {
			return mid, true
		} else if ids[mid].Less(target) {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return right, false
}
