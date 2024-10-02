package schedule

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	"fmt"
	"sync"
)

type TaskWrapperNode struct {
	TaskWrapper
	Next *TaskWrapperNode

	//TODO: Add ExecCtx to ProcessorTree
}

type listEft struct {
	prev *TaskWrapperNode
	eft  uint64
}

func (l *listEft) IsLessThan(e eftResult) bool {
	return e == nil || l.EFT() < e.EFT()
}

func (l *listEft) EFT() uint64 {
	return l.eft
}

type ProcessorList struct {
	head     *TaskWrapperNode
	execCtx  *eutils.ExecContext
	wg       *sync.WaitGroup
	totalGas uint64
}

func NewProcessorList() *ProcessorList {
	return &ProcessorList{
		head: &TaskWrapperNode{
			TaskWrapper: TaskWrapper{
				Task:     nil,
				Priority: 0,
				EST:      0,
				EFT:      0,
			},
			Next: nil,
		},
	}
}

func (pl *ProcessorList) SetExecCtx(execCtx *eutils.ExecContext, wg *sync.WaitGroup) {
	pl.execCtx = execCtx
	pl.wg = wg
}

func (pl *ProcessorList) Print() {
	cur := pl.head.Next
	for cur != nil {
		fmt.Print(cur.TaskWrapper.Task.Tid, " ")
		cur = cur.Next
	}
	fmt.Println()
}

func (pl *ProcessorList) SetTimespan(timespan uint64) {}

func (pl *ProcessorList) FindEFT(tw *TaskWrapper) eftResult {
	cur := pl.head
	var prev *TaskWrapperNode
	for cur.Next != nil {
		if cur.Next.EST >= tw.Task.Cost+max(cur.EFT, tw.EST) {
			prev = cur
			break
		}
		cur = cur.Next
	}

	if cur.Next == nil {
		prev = cur
	}

	return &listEft{
		prev: prev,
		eft:  max(prev.EFT, tw.EST) + tw.Task.Cost,
	}
}

func (pl *ProcessorList) AddTask(tw *TaskWrapper, e eftResult) {
	listeft, ok := e.(*listEft)
	if !ok {
		panic("AddTask: e is not of type *listEft")
	}
	prev := listeft.prev
	if prev == nil {
		panic("AddTask: prev is nil")
	}
	originNext := prev.Next
	prev.Next = &TaskWrapperNode{
		TaskWrapper: *tw,
		Next:        originNext,
	}
}

// execute the task in the processor list
func (pl *ProcessorList) Execute() {
	defer pl.wg.Done()
	evm := vm.NewEVM(pl.execCtx.BlockCtx, evmtypes.TxContext{}, pl.execCtx.ExecState, pl.execCtx.ChainCfg, vm.Config{})
	for cur := pl.head.Next; cur != nil; cur = cur.Next {
		task := cur.Task
		if task.Msg == nil {
			continue
		}
		msg := task.Msg
		if pl.execCtx.EarlyAbort {
			pl.execCtx.SetTask(task, nil)
		} else {
			newRwSet := rwset.NewRwSet()
			pl.execCtx.SetTask(task, newRwSet)
		}
		evm.TxContext = pl.execCtx.TxCtx
		task.Wait() // waiting for the task to be ready
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			pl.totalGas += res.UsedGas
		}
		pl.execCtx.ExecState.Commit()
	}
}

func (pl *ProcessorList) GetGas() uint64 {
	return pl.totalGas
}
