package orm

import (
	"github.com/btcboost/copernicus/orm/boltdb"
	"github.com/btcboost/copernicus/orm/database"
)

type DriverType int

const (
	_ DriverType = iota
	DBBolt
)

func InitDB(driverType DriverType, path string) (database.DBBase, error) {
	var db database.DBBase
	if driverType == DBBolt {
		db, err := boltdb.NewBlotDB(path)
		return db, err
	}
	return db, nil
}
