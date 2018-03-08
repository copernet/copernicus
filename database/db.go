package database

import "github.com/btcboost/copernicus/database/boltdb"

type DriverType int

const (
	_ DriverType = iota
	DBBolt
)

const (
	DB_COIN    = 'C'
	DB_COINS   = 'c'
	DB_TXINDEX = 't'

	DB_BEST_BLOCK  = 'B'
	DB_BLOCK_INDEX = 'b'
	DB_BLOCK_FILES = 'f'

	DB_REINDEX_FLAG = 'R'
)

func InitDB(driverType DriverType, path string) (boltdb.DBBase, error) {
	var db boltdb.DBBase
	if driverType == DBBolt {
		db, err := boltdb.NewBlotDB(path)
		return db, err
	}
	return db, nil
}
