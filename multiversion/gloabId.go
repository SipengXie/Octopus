package multiversion

type GlobalId struct {
	BlockNumber uint64
	TxIndex     int
	Incarnation int // used for re-executed transaction
}

const MAXUINT64 = ^uint64(0)

var SnapshotID = &GlobalId{BlockNumber: 0, TxIndex: -1, Incarnation: -1}
var EndID = &GlobalId{BlockNumber: MAXUINT64, TxIndex: -1, Incarnation: -1}

func NewGlobalId(blockNumber uint64, txIndex int, incarnation int) *GlobalId {
	return &GlobalId{
		BlockNumber: blockNumber,
		TxIndex:     txIndex,
		Incarnation: incarnation,
	}
}

func (g *GlobalId) Equal(other *GlobalId) bool {
	return g.BlockNumber == other.BlockNumber && g.TxIndex == other.TxIndex && g.Incarnation == other.Incarnation
}

func (g *GlobalId) LessThan(other *GlobalId) bool {
	if g.BlockNumber < other.BlockNumber {
		return true
	}
	if g.BlockNumber == other.BlockNumber {
		if g.TxIndex == other.TxIndex {
			return g.Incarnation < other.Incarnation
		} else {
			return g.TxIndex < other.TxIndex
		}
	}
	return false
}
