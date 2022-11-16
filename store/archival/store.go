package archival

import (
	dbm "github.com/tendermint/tm-db"
)

type ReadOnlyCommitKVStore struct {
	db dbm.DB
}
