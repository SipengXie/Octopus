package treeutils

const MAXUINT64 = ^uint64(0) >> 1

type segTreeNode struct {
	L, R       uint64
	seg_max    uint64
	lson, rson *segTreeNode
}

func NewNode(L, R uint64) *segTreeNode {
	return &segTreeNode{L: L, R: R}
}

type segTree struct {
	root *segTreeNode
}

func NewSegTree(L, R uint64) *segTree {
	root := NewNode(L, R)
	return &segTree{root: root}
}

func (t *segTree) Modify(x, val uint64) {
	t.modify(t.root, x, val)
}

func (t *segTree) modify(cur *segTreeNode, x, val uint64) {
	if cur.L == cur.R {
		cur.seg_max = val
		return
	}
	mid := (cur.L + cur.R) >> 1
	if x <= mid {
		if cur.lson == nil {
			cur.lson = NewNode(cur.L, mid)
		}
		t.modify(cur.lson, x, val)
	} else {
		if cur.rson == nil {
			cur.rson = NewNode(mid+1, cur.R)
		}
		t.modify(cur.rson, x, val)
	}
	if cur.lson == nil {
		cur.seg_max = cur.rson.seg_max
	} else if cur.rson == nil {
		cur.seg_max = cur.lson.seg_max
	} else {
		cur.seg_max = max(cur.lson.seg_max, cur.rson.seg_max)
	}

}

func (t *segTree) Query(L, R, threshold uint64) (uint64, uint64) {
	return t.query(t.root, L, R, threshold)
}

func (t *segTree) query(cur *segTreeNode, L, R, threshold uint64) (uint64, uint64) {
	if cur.seg_max < threshold {

		return MAXUINT64, 0
	}
	if cur.L == cur.R {
		return cur.L, cur.seg_max
	}
	mid := (cur.L + cur.R) >> 1

	if R <= mid {
		if cur.lson != nil {
			return t.query(cur.lson, L, R, threshold)
		}

		return MAXUINT64, 0
	}

	if L > mid {
		if cur.rson != nil {
			return t.query(cur.rson, L, R, threshold)
		}

		return MAXUINT64, 0
	}

	var ans uint64 = MAXUINT64
	var ansLength uint64 = 0
	if cur.lson != nil {
		ans, ansLength = t.query(cur.lson, L, mid, threshold)
	}
	if ans != MAXUINT64 {

		return ans, ansLength
	}

	if cur.rson != nil {
		return t.query(cur.rson, mid+1, R, threshold)
	}
	return MAXUINT64, 0
}
