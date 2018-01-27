package boltdb

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/btcboost/copernicus/orm"
)

type BoltDB struct {
	orm.DBBase
	*bolt.DB
	filePath string
}

func NewBlotDB() orm.DBBase {
	boltdb := new(BoltDB)
	return boltdb
}

func (boltdb *BoltDB) Type() string {
	return "boltdb"
}

func (boltdb *BoltDB) Begin(writable bool) (orm.DBTx, error) {
	return nil, nil

}

func (boltdb *BoltDB) View(fn func(tx orm.DBTx) error) error {
	return nil
}

func (boltdb *BoltDB) Update(fn func(tx orm.DBTx) error) error {
	return nil
}

func (boltdb *BoltDB) Open() error {
	db, err := bolt.Open(boltdb.filePath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	boltdb.DB = db
	return nil
}

func (boltdb *BoltDB) Close() error {
	return boltdb.DB.Close()
}
