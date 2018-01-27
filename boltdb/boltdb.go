package boltdb

import (
	"github.com/btcboost/copernicus/orm"
)

type BoltDB struct {
	orm.DBBase
}

func NewBlotDB() orm.DBBase {
	db := new(BoltDB)
	return db
}

func (db *BoltDB) Type() string {
	return "boltdb"
}

func (db *BoltDB) Begin(writable bool) (orm.DBTx, error) {
	return nil, nil

}

func (db *BoltDB) View(fn func(tx orm.DBTx) error) error {
	return nil
}

func (db *BoltDB) Update(fn func(tx orm.DBTx) error) error {
	return nil
}

func (db *BoltDB) Close() error {
	return nil
}
