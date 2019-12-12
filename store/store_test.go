package store

import (
	"testing"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
)

func Test_CommitDynamicMultiStore(t *testing.T) {
	db := dbm.NewMemDB()
	rootStore := NewCommitMultiStore(db)

	rootStore.MountStoreWithDB(sdk.NewKVStoreKey("bank"), sdk.StoreTypeIAVL, nil)
	_ = rootStore.LoadLatestVersion()

	// keeps last 1 + every 10th
	rootStore.SetPruning(store.NewPruningOptions(1, 10))

	// commit 15 times
	for i := 0; i < 15; i++ {
		rootStore.Commit()
	}

	// add a new store
	rootStore.MountStoreWithDB(sdk.NewKVStoreKey("staking"), sdk.StoreTypeIAVL, nil)
	_ = rootStore.LoadLatestVersion()

	// commit 10 times
	for i := 0; i < 10; i++ {
		rootStore.Commit()
	}

	require.Equal(t, int64(25), rootStore.LastCommitID().Version)

	err := rootStore.LoadVersion(20)
	require.NoError(t, err)
}
