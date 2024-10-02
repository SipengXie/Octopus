package pipeline

import (
	mv "blockConcur/multiversion"
	"blockConcur/rwset"
	"blockConcur/state"
	"blockConcur/types"
	"blockConcur/utils"
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

func GenerateAccessedBy(tasks []*types.Task) *rwset.RwAccessedBy {
	rwAccessedBy := rwset.NewRwAccessedBy()
	for _, task := range tasks {
		rwAccessedBy.Add(task.RwSet, task.Tid)
	}
	return rwAccessedBy
}

// Prefetcher fetch the corresponding data to the cache, and generate the accessedBy map.
type Prefetcher struct {
	cache      *state.MvCache
	FetchPool  *ants.PoolWithFunc
	IVPool     *ants.PoolWithFunc
	Wg         *sync.WaitGroup
	InputChan  chan *TaskMessage
	OutputChan chan *BuildGraphMessage
}

type keyAndWg struct {
	key string
	wg  *sync.WaitGroup
}

type taskAndWg struct {
	task *types.Task
	wg   *sync.WaitGroup
}

func GeneratePools(cache *state.MvCache, fetchPoolSize, ivPoolSize int) (fetchPool, ivPool *ants.PoolWithFunc) {
	// generate a prefetch fetchPool, each thread prefetch one task.
	// Fetch the read_set of the task into cache.
	fetchPool, err := ants.NewPoolWithFunc(fetchPoolSize, func(i interface{}) {
		// i is a struct of key and waitGroup
		taskAndWg := i.(*keyAndWg)
		key := taskAndWg.key
		wg := taskAndWg.wg
		defer wg.Done()
		cache.Fetch(utils.ParseKey(key))
	})
	if err != nil {
		panic(err)
	}
	// generate a task handling pool, adding inital read versions to the task.
	// generate the initial write versions, and install them to the cache.
	ivPool, err = ants.NewPoolWithFunc(ivPoolSize, func(i interface{}) {
		taskAndWg := i.(*taskAndWg)
		task := taskAndWg.task
		wg := taskAndWg.wg
		defer wg.Done()
		// adding task.rwset.read_set to task.ReadVersions
		for key := range task.RwSet.ReadSet {
			// TODO: if we change this version to the last block's last inserted version,
			// we could achieve inter-block concurrency control. However, we have
			// two problems:
			// (1) the block root generation problem.
			// (2) prefetch_1 -> graph_1 -> prefetch_2 -> graph_2, the prefetch_2 should
			// happen after graph_1. This problem does not exist as prefetch_2 is happend after prefetch_1.
			// The MVCache contains all versions generated in the previous block.
			// Now, no need for inter-block concurrency.
			v := cache.GetCommittedVersion(key)
			task.AddReadVersion(key, v)
		}
		// adding task.rwset.write_set to task.WriteVersions and install them to the cache.
		for key := range task.RwSet.WriteSet {
			v := mv.NewVersion(nil, task.Tid, mv.Pending)
			cache.InsertVersion(key, v)
			task.AddWriteVersion(key, v)
		}
	})
	if err != nil {
		panic(err)
	}
	return
}

func NewPrefetcher(cache *state.MvCache, wg *sync.WaitGroup, fetchPoolSize, ivPoolSize int, in chan *TaskMessage, out chan *BuildGraphMessage) *Prefetcher {
	fetchPool, ivPool := GeneratePools(cache, fetchPoolSize, ivPoolSize)
	return &Prefetcher{
		cache:      cache,
		FetchPool:  fetchPool,
		IVPool:     ivPool,
		Wg:         wg,
		InputChan:  in,
		OutputChan: out,
	}
}

func Prefetch(tasks types.Tasks, fetchPool, ivPool *ants.PoolWithFunc) (float64, *rwset.RwAccessedBy) {
	// TODO: This two function could be merged...
	rwAccessedBy := GenerateAccessedBy(tasks)
	// Parallel prefetch the keys in rwAccessedBy's readBy map
	st := time.Now()
	var wg sync.WaitGroup
	for key := range rwAccessedBy.ReadBy {
		wg.Add(1)
		fetchPool.Invoke(&keyAndWg{key: key, wg: &wg})
	}
	wg.Wait()
	cost := time.Since(st).Seconds()

	// Parallel add initial read/write versions to the tasks
	for _, task := range tasks {
		wg.Add(1)
		ivPool.Invoke(&taskAndWg{task: task, wg: &wg})
	}
	wg.Wait()
	return cost, rwAccessedBy
}

func (g *Prefetcher) Run() {
	var elapsed float64
	for input := range g.InputChan {

		if input.Flag == END {
			outMessage := &BuildGraphMessage{
				Flag: END,
			}
			g.OutputChan <- outMessage
			close(g.OutputChan)
			g.FetchPool.Release()
			g.Wg.Done()
			fmt.Println("Prefetch Cost:", elapsed, "ms")
			return
		}

		tasks := input.Tasks

		cost, rwAccessedBy := Prefetch(tasks, g.FetchPool, g.IVPool)
		elapsed += cost

		outMessage := &BuildGraphMessage{
			Flag:         START,
			Tasks:        tasks,
			RwAccessedBy: rwAccessedBy,
		}
		g.OutputChan <- outMessage
	}
}
