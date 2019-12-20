package store_test

import (
	"testing"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	store "github.com/cosmos/cosmos-sdk/store/types"
	nftsimapp "github.com/cosmos/modules/incubator/nft/app"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

func Test_AddModule(t *testing.T) {
	// keeps last 1 + every 10th
	pruningOpt := bam.SetPruning(store.NewPruningOptions(1, 10))

	db := dbm.NewMemDB()

	// new sdk.simapp
	app := NewApp(db, pruningOpt)

	header := abci.Header{
		AppHash: []byte("apphash"),
		Height:  0,
	}

	// commit 15 times
	for i := 0; i < 15; i++ {
		header.Height = header.Height + 1
		app.BeginBlock(abci.RequestBeginBlock{Header: header})
		app.Commit()
	}
	require.Equal(t, int64(15), app.LastBlockHeight())

	// nftApp add a new module - nft
	nftApp := NewNftApp(db, pruningOpt)

	// commit 10 times
	for i := 0; i < 10; i++ {
		header.Height = header.Height + 1
		nftApp.BeginBlock(abci.RequestBeginBlock{Header: header})
		nftApp.Commit()
	}
	require.Equal(t, int64(25), nftApp.LastBlockHeight())

	// load to block height 20
	err := nftApp.LoadHeight(20)
	require.Equal(t,
		"failed to load Store: wanted to load target 5 but only found up to 0",
		err.Error())
	//When loading the nft store target to block height 20, its version is 5, but this version had been pruned.
}

func NewApp(db dbm.DB, pruningOpt func(*bam.BaseApp)) *simapp.SimApp {
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, 0, pruningOpt)

	genesisState := simapp.NewDefaultGenesisState()
	stateBytes, err := codec.MarshalJSONIndent(app.Codec(), genesisState)
	if err != nil {
		panic(err)
	}

	// Initialize the chain
	app.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	return app
}

func NewNftApp(db dbm.DB, pruningOpt func(*bam.BaseApp)) *nftsimapp.SimApp {
	app := nftsimapp.NewSimApp(log.NewNopLogger(), db, nil, true, 0, pruningOpt)
	return app
}
