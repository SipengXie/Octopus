package schedule

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	utils "blockConcur/schedule/tree_utils"
	"blockConcur/state"
	"blockConcur/types"
	"container/heap"
	"fmt"
	"sync"
)

type treeEftResult struct {
	slot *utils.Slot
	eft  uint64
}

func (ter *treeEftResult) IsLessThan(other eftResult) bool {
	return other == nil || ter.EFT() < other.EFT()
}

func (ter *treeEftResult) EFT() uint64 {
	return ter.eft
}

type ProcessorTree struct {
	Tasks       ASTTaskQueue
	SlotManager *utils.SlotManager

	execCtx      *eutils.ExecContext
	wg           *sync.WaitGroup
	totalGas     uint64
	size         int
	deferedTasks types.Tasks
}

func NewProcessorTree() *ProcessorTree {
	return &ProcessorTree{
		Tasks: make(ASTTaskQueue, 0),
	}
}

func (pt *ProcessorTree) SetExecCtx(execCtx *eutils.ExecContext, wg *sync.WaitGroup) {
	pt.execCtx = execCtx
	pt.wg = wg
}

func (pt *ProcessorTree) Print() {
	for _, task := range pt.Tasks {
		fmt.Print(task.Task.Tid, " ")
	}
	fmt.Println()
}

func (pt *ProcessorTree) SetTimespan(timespan uint64) {
	pt.SlotManager = utils.NewSlotsManager(timespan)
}

func (pt *ProcessorTree) FindEFT(task *TaskWrapper) eftResult {
	slot := pt.SlotManager.FindSlot(task.EST, task.Task.Cost)
	res := &treeEftResult{
		slot: slot,
	}
	if slot.St <= task.EST {
		res.eft = task.EST + task.Task.Cost
	} else {
		res.eft = slot.St + task.Task.Cost
	}
	return res
}

func (pt *ProcessorTree) AddTask(task *TaskWrapper, eft eftResult) {
	heap.Push(&pt.Tasks, task)
	tRes, ok := eft.(*treeEftResult)
	if !ok {
		panic("AddTask: invalid eft type, should be treeEftResult")
	}
	slot := tRes.slot
	if slot.St <= task.AST {
		prevSlot := &utils.Slot{St: tRes.slot.St, Length: task.AST - tRes.slot.St}
		newSlot := &utils.Slot{St: task.EFT, Length: slot.St + slot.Length - task.EFT}
		pt.SlotManager.ModifySlot(prevSlot)
		pt.SlotManager.AddSlot(newSlot)
	} else {
		prevSlot := &utils.Slot{St: tRes.slot.St, Length: 0}
		newSlot := &utils.Slot{St: task.EFT, Length: slot.Length - task.Task.Cost}
		pt.SlotManager.ModifySlot(prevSlot)
		pt.SlotManager.AddSlot(newSlot)
	}
	pt.size++
}

func (pt *ProcessorTree) Size() int {
	return pt.size
}

func (pt *ProcessorTree) Execute() {
	defer pt.wg.Done()
	evm := vm.NewEVM(pt.execCtx.BlockCtx, evmtypes.TxContext{}, pt.execCtx.ExecState, pt.execCtx.ChainCfg, vm.Config{})
	deferedTasks := make(types.Tasks, 0)
	for pt.Tasks.Len() > 0 {
		task := heap.Pop(&pt.Tasks).(*TaskWrapper).Task
		if task.Msg == nil {
			continue
		}
		msg := task.Msg
		var newRwSet *rwset.RwSet
		if pt.execCtx.EarlyAbort {
			pt.execCtx.SetTask(task, nil)
		} else {
			newRwSet = rwset.NewRwSet()
			pt.execCtx.SetTask(task, newRwSet)
		}
		evm.TxContext = pt.execCtx.TxCtx
		// task.Wait() // waiting for the task to be ready
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			pt.totalGas += res.UsedGas
		} else if _, ok := err.(*state.InvalidError); ok {
			deferedTasks = append(deferedTasks, task)
		}
		if newRwSet != nil {
			task.RwSet = newRwSet
		}
		pt.execCtx.ExecState.Commit()
	}
	pt.deferedTasks = deferedTasks
}

func (pt *ProcessorTree) GetGas() uint64 {
	return pt.totalGas
}

func (pt *ProcessorTree) GetDeferedTasks() types.Tasks {
	return pt.deferedTasks
}
