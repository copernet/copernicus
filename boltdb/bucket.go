package boltdb

import (
	"github.com/btcboost/copernicus/orm"
	"github.com/boltdb/bolt"
)

type bucket struct {
	orm.Bucket
	boltBucket bolt.Bucket
}

func (bucket *bucket) ForEach(func(k, v []byte) error) error {
	return nil
}

func (bucket *bucket) Cursor() orm.Cursor {
	return nil
}

func (bucket *bucket) Writable() bool {
	return false
}

func (bucket *bucket) Put(key, value []byte) error {
	return nil
}

func (bucket *bucket) Get(key []byte) []byte {
	return nil
}

func (bucket *bucket) Delete(key [] byte) error {
	return nil
}
