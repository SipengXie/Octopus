package schedule

import (
	"blockConcur/eutils"
	"blockConcur/types"
	"sync"
)

type eftResult interface {
	IsLessThan(eftResult) bool
	EFT() uint64
}

type Processor interface {
	SetTimespan(uint64)
	FindEFT(*TaskWrapper) eftResult
	AddTask(*TaskWrapper, eftResult)
	Print()
	Size() int
	// TODO: complete the function
	Execute()
	SetExecCtx(*eutils.ExecContext, *sync.WaitGroup)
	GetGas() uint64
	GetDeferedTasks() types.Tasks
}

type Processors []Processor
