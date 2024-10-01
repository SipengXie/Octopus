package helper

import (
	innerstate "blockConcur/state/inner_state"
	"context"
	"sync"

	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/chain/snapcfg"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/mdbx"
	state2 "github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/systemcontracts"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/eth/ethconfig"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/turbo/snapshotsync/freezeblocks"
	"github.com/ledgerwatch/log/v3"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/semaphore"
)

const PATH = "/chaindata/erigondata/chaindata"
const LABEL = kv.ChainDB
const SNAPSHOT = "/chaindata/erigondata/snapshots"
const ThreadsLimit = 9_000

func dbCfg(label kv.Label, path string) mdbx.MdbxOpts {
	limiterB := semaphore.NewWeighted(ThreadsLimit)
	opts := mdbx.NewMDBX(log.New()).Path(path).Label(label).RoTxsLimiter(limiterB)
	opts = opts.Accede()
	return opts
}

func openDB() kv.RoDB {
	db := dbCfg(LABEL, PATH).MustOpen()
	return db
}

func newBlockReader(cfg ethconfig.Config) *freezeblocks.BlockReader {
	var minFrozenBlock uint64

	if frozenLimit := cfg.Sync.FrozenBlockLimit; frozenLimit != 0 {
		if maxSeedable := snapcfg.MaxSeedableSegment(cfg.Genesis.Config.ChainName, SNAPSHOT); maxSeedable > frozenLimit {
			minFrozenBlock = maxSeedable - frozenLimit
		}
	}

	blockSnaps := freezeblocks.NewRoSnapshots(cfg.Snapshot, SNAPSHOT, minFrozenBlock, log.New())
	borSnaps := freezeblocks.NewBorRoSnapshots(cfg.Snapshot, SNAPSHOT, minFrozenBlock, log.New())
	blockSnaps.ReopenFolder()
	borSnaps.ReopenFolder()
	return freezeblocks.NewBlockReader(blockSnaps, borSnaps)
}

type GloablEnv struct {
	Ctx         context.Context
	BlockReader *freezeblocks.BlockReader
	DB          kv.RoDB
	Cfg         *chain.Config
	// 256 header cache, header inside is continous
	Headers [256]*types.Header
}

func PrepareEnv() GloablEnv {
	// consoleHandler := log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
	// log.Root().SetHandler(consoleHandler)
	log.Info("Starting")
	ctx := context.Background()

	cfg := ethconfig.Defaults
	db := openDB()
	log.Info("DB opened")
	blockReader := newBlockReader(cfg)
	log.Info("Block Reader created")

	return GloablEnv{
		Ctx:         ctx,
		BlockReader: blockReader,
		DB:          db,
		Cfg:         params.MainnetChainConfig,
		Headers:     [256]*types.Header{},
	}
}

func (g *GloablEnv) GetBlockAndHeader(blockNumber uint64) (*types.Block, *types.Header) {
	dbTx, err := g.DB.BeginRo(g.Ctx)
	if err != nil {
		panic(err)
	}
	defer dbTx.Rollback()

	blk, err := g.BlockReader.BlockByNumber(g.Ctx, dbTx, blockNumber)
	if err != nil {
		panic(err)
	}
	header, err := g.BlockReader.Header(g.Ctx, dbTx, blk.Hash(), blk.NumberU64())
	if err != nil {
		panic(err)
	}

	// fetch corresponding 256 headers in parallel (for simplicity)
	// decalring dbTxs
	dbTxs := make([]kv.Tx, 256)
	for i := range dbTxs {
		dbTxs[i], err = g.DB.BeginRo(g.Ctx)
		if err != nil {
			panic(err)
		}
	}

	var wg sync.WaitGroup
	pool, _ := ants.NewPoolWithFunc(256, func(i interface{}) {
		defer wg.Done()
		// fetch header
		idx := i.(int)
		header, err := g.BlockReader.HeaderByNumber(g.Ctx, dbTxs[idx], blk.NumberU64()-uint64(256-idx))
		if err != nil {
			panic(err)
		}
		g.Headers[idx] = header
	})

	for i := range dbTxs {
		wg.Add(1)
		pool.Invoke(i)
	}
	wg.Wait()

	return blk, header
}

func (g *GloablEnv) GetIBS(blockNumber uint64, dbTx kv.Tx) *innerstate.IntraBlockState {
	pls := state2.NewPlainState(dbTx, blockNumber, systemcontracts.SystemContractCodeLookup[g.Cfg.ChainName])
	ibs := innerstate.New(pls)
	return ibs
}

// func (g *GloablEnv) NewExecContext(header *types.Header) *types2.ExecContext {
// 	return types2.NewExecContext(g.BlockReader, header, g.Cfg, g.Headers)
// }
