package schedule

import (
	"blockConcur/eutils"
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
	// TODO: complete the function
	Execute()
	SetExecCtx(*eutils.ExecContext, *sync.WaitGroup)
}

type Processors []Processor
