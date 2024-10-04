package schedule

import (
	"blockConcur/eutils"
	"fmt"
	"sync"
)

type simpleEftResult struct {
	eft uint64
}

func (s *simpleEftResult) IsLessThan(eftResult eftResult) bool {
	return eftResult == nil || s.EFT() < eftResult.EFT()
}

func (s *simpleEftResult) EFT() uint64 {
	return s.eft
}

type ProcessorSimple struct {
	Tasks []*TaskWrapper
}

func NewProcessorSimple() *ProcessorSimple {
	return &ProcessorSimple{
		Tasks: make([]*TaskWrapper, 0),
	}
}

func (p *ProcessorSimple) SetExecCtx(execCtx *eutils.ExecContext, wg *sync.WaitGroup) {
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

func (p *ProcessorSimple) Execute() {}

func (p *ProcessorSimple) GetGas() uint64 {
	return 0
}
