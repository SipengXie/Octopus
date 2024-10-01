package treeutils

type Slot struct {
	St     uint64
	Length uint64
}

type SlotManager struct {
	Slots       *avlBST
	TimeSpan    *segTree
	largestTime uint64
}

func NewSlotsManager(timespan uint64) *SlotManager {
	bst := NewTree()
	bst.Add(0, timespan<<1)
	segt := NewSegTree(0, timespan<<1)
	segt.Modify(0, timespan<<1)
	return &SlotManager{
		Slots:       bst,
		TimeSpan:    segt,
		largestTime: timespan << 1,
	}
}

func (sm *SlotManager) FindSlot(EST, length uint64) *Slot {

	node := sm.Slots.FindMaxLessThan(EST)
	if node != nil && node.st+node.length >= EST+length {
		return &Slot{node.st, node.length}
	}

	st, len := sm.TimeSpan.Query(EST, sm.largestTime, length)
	if st == MAXUINT64 {
		panic("findSlot: no slot found")
	}
	return &Slot{st, len}
}

func (sm *SlotManager) AddSlot(s *Slot) {
	if s.Length != 0 {
		sm.Slots.Add(s.St, s.Length)
	}
	sm.TimeSpan.Modify(s.St, s.Length)
}

// Modify Slot with the input slot, the original slot must has the same key as the input slot
// TODO: we did not check the key of the input slot here.
func (sm *SlotManager) ModifySlot(s *Slot) {
	sm.Slots.Remove(s.St)
	sm.AddSlot(s)
}
