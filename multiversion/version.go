package multiversion

import (
	"sync"
)

type Status int

const (
	Pending Status = iota
	Committed
	Ignore
)

type Version struct {
	Data   interface{}
	Tid    *GlobalId
	Status Status
	Next   *Version
	Prev   *Version

	Plock sync.Mutex
	Nlock sync.Mutex

	Readby    map[int]struct{}
	MaxReadby int

	Mu   sync.Mutex
	Cond *sync.Cond
}

func NewVersion(data interface{}, tid *GlobalId, status Status) *Version {
	v := &Version{
		Data:      data,
		Tid:       tid,
		Status:    status,
		Readby:    make(map[int]struct{}),
		MaxReadby: -1,
		Next:      nil,
		Prev:      nil,
		Plock:     sync.Mutex{},
		Nlock:     sync.Mutex{},
		Mu:        sync.Mutex{},
	}
	v.Cond = sync.NewCond(&v.Mu)
	return v
}

func (v *Version) InsertOrNext(iv *Version) *Version {
	v.Nlock.Lock()
	defer v.Nlock.Unlock()
	if v.Next == nil || v.updatePrev(iv) {
		iv.Next = v.Next
		v.Next = iv
		iv.Prev = v
		return nil
	} else {
		return v.Next
	}
}

func (v *Version) updatePrev(iv *Version) bool {
	v.Plock.Lock()
	defer v.Plock.Unlock()
	if iv.Tid.LessThan(v.Tid) {
		v.Prev = iv
		return true
	}
	return false
}

func (v *Version) GetVisible() *Version {
	if v.Status != Committed {
		return v.Prev.GetVisible()
	}
	return v
}
