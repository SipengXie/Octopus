package schedule

import (
	"blockConcur/eutils"
	core "blockConcur/evm"
	"blockConcur/evm/vm"
	"blockConcur/evm/vm/evmtypes"
	"blockConcur/rwset"
	"blockConcur/types"
	"fmt"
	"sync"
)

type simpleEftResult struct {
	eft uint64
}

func (s *simpleEftResult) IsLessThan(eftResult eftResult) bool {
	return eftResult == nil || s.EFT() < eftResult.EFT()
}

func (s *simpleEftResult) Equals(eftResult eftResult) bool {
	return s.EFT() == eftResult.EFT()
}

func (s *simpleEftResult) EFT() uint64 {
	return s.eft
}

type ProcessorSimple struct {
	Tasks        []*TaskWrapper
	execCtx      *eutils.ExecContext
	wg           *sync.WaitGroup
	totalGas     uint64
	deferedTasks types.Tasks
}

func NewProcessorSimple() *ProcessorSimple {
	return &ProcessorSimple{
		Tasks: make([]*TaskWrapper, 0),
	}
}

func (p *ProcessorSimple) SetExecCtx(execCtx *eutils.ExecContext, wg *sync.WaitGroup) {
	p.execCtx = execCtx
	p.wg = wg
}

func (p *ProcessorSimple) Print() {
	for _, task := range p.Tasks {
		fmt.Print(task.Task.Tid, " ")
	}
	fmt.Println()
}

func (p *ProcessorSimple) SetTimespan(ts uint64) {
}

func (p *ProcessorSimple) FindEFT(tw *TaskWrapper) eftResult {
	len := len(p.Tasks)
	if len == 0 { // If there are no tasks, return 0
		return &simpleEftResult{tw.EST + tw.Task.Cost}
	} else {
		return &simpleEftResult{max(tw.EST+tw.Task.Cost, p.Tasks[len-1].EFT+tw.Task.Cost)}
	}
}

func (p *ProcessorSimple) AddTask(tw *TaskWrapper, eftResult eftResult) {
	p.Tasks = append(p.Tasks, tw)
}

func (p *ProcessorSimple) Size() int {
	return len(p.Tasks)
}

func (p *ProcessorSimple) Execute() {
	defer p.wg.Done()
	evm := vm.NewEVM(p.execCtx.BlockCtx, evmtypes.TxContext{}, p.execCtx.ExecState, p.execCtx.ChainCfg, vm.Config{})
	deferedTasks := make(types.Tasks, 0)
	for _, tw := range p.Tasks {
		task := tw.Task
		if task.Msg == nil {
			continue
		}
		msg := task.Msg
		var newRwSet *rwset.RwSet
		if !p.execCtx.EarlyAbort {
			newRwSet = rwset.NewRwSet()
		}
		p.execCtx.SetTask(task, newRwSet)
		evm.TxContext = p.execCtx.TxCtx

		// task.Wait() // waiting for the task to be ready
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.Gas()).AddBlobGas(msg.BlobGas()), true /* refunds */, false /* gasBailout */)
		if err == nil {
			p.totalGas += res.UsedGas
		} else {
			if p.execCtx.ExecState.GetReadIgnored() {
				p.execCtx.ExecState.MarkDefered()
			}
		}

		if newRwSet != nil {
			task.RwSet = newRwSet
		}
		committed := p.execCtx.ExecState.Commit()
		if !committed {
			deferedTasks = append(deferedTasks, task)
		}
	}
	p.deferedTasks = deferedTasks
}
func (p *ProcessorSimple) GetGas() uint64 {
	return p.totalGas
}

func (p *ProcessorSimple) GetDeferedTasks() types.Tasks {
	return p.deferedTasks
}
