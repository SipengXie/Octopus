package schedule

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
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
	size     int
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
	pl.size++
}

func (pl *ProcessorList) Size() int {
	return pl.size
}

// execute the task in the processor list
func (pl *ProcessorList) Execute() {
	defer pl.wg.Done()
	evm := vm.NewEVM(pl.execCtx.BlockCtx, evmtypes.TxContext{}, pl.execCtx.ExecState, pl.execCtx.ChainCfg, vm.Config{})
	deferedTasks := make(types.Tasks, 0)
	for cur := pl.head.Next; cur != nil; cur = cur.Next {
		task := cur.Task
		if task.Msg == nil {
			continue
		}
		msg := task.Msg
		var newRwSet *rwset.RwSet
		if pl.execCtx.EarlyAbort {
			pl.execCtx.SetTask(task, nil)
		} else {
			newRwSet = rwset.NewRwSet()
			pl.execCtx.SetTask(task, newRwSet)
		}
		evm.TxContext = pl.execCtx.TxCtx

		/* This code section is used for debugging
		var tracer vm.EVMLogger
		var evm *vm.EVM
		if task.TxHash == common.HexToHash("0x83d6a34cf13f93bc418ceb5ced9b61f640a3e936fbd98f6d8c6d4896ab70d12b") {
			tracer = helper.NewStructLogger(&helper.LogConfig{})
			evm = vm.NewEVM(pl.execCtx.BlockCtx, pl.execCtx.TxCtx, pl.execCtx.ExecState, pl.execCtx.ChainCfg, vm.Config{Debug: true, Tracer: tracer})
		} else {
			evm = vm.NewEVM(pl.execCtx.BlockCtx, pl.execCtx.TxCtx, pl.execCtx.ExecState, pl.execCtx.ChainCfg, vm.Config{})
		}
		*/

		task.Wait() // waiting for the task to be ready
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			pl.totalGas += res.UsedGas
		} else if _, ok := err.(*state.InvalidError); ok {
			deferedTasks = append(deferedTasks, task)
		}

		/* This code section is used for debugging
		if task.TxHash == common.HexToHash("0x83d6a34cf13f93bc418ceb5ced9b61f640a3e936fbd98f6d8c6d4896ab70d12b") {
			if structLogs, ok := tracer.(*helper.StructLogger); ok {
				structLogs.Flush(task.TxHash)
			}
		}
		*/
		if newRwSet != nil {
			task.RwSet = newRwSet
		}
		pl.execCtx.ExecState.Commit()
	}
	pl.totalGas += processDeferedTasks(deferedTasks, pl.execCtx, evm, pl.execCtx.EarlyAbort)
}

func (pl *ProcessorList) GetGas() uint64 {
	return pl.totalGas
}

// process the defered tasks
// if early_abort is true, we will serial execute the defered tasks (tasks do not carry out the rwset)
// TODO: if early_abort is false, we will parallel execute the defered tasks with blockConcur, which can handle the inaccurate rwset problem
func processDeferedTasks(deferedTasks types.Tasks, execCtx *eutils.ExecContext, evm *vm.EVM, is_serial bool) uint64 {
	totalGas := uint64(0)
	for _, task := range deferedTasks {
		// give task a new ID, the incarnation number will be set to 1
		task.Tid = utils.NewID(task.Tid.BlockNumber, task.Tid.TxIndex, 1)
		msg := task.Msg
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			execCtx.ExecState.Commit()
			totalGas += res.UsedGas
		}
	}
	return totalGas
}
