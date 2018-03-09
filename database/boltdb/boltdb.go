package boltdb

import (
	"time"

	"github.com/boltdb/bolt"
)

type BoltDB struct {
	DBBase
	*bolt.DB
	filePath string
}

func NewBlotDB(filepath string) (DBBase, error) {
	boltdb := new(BoltDB)
	boltdb.filePath = filepath
	err := boltdb.Open()
	if err != nil {
		return nil, err
	}
	return boltdb, nil
}

func (boltdb *BoltDB) Type() string {
	return "boltdb"
}

func (boltdb *BoltDB) View(key []byte, fn func(bucket Bucket) error) error {
	err := boltdb.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(key)
		bucket := new(bucket)
		bucket.boltBucket = b
		err := fn(bucket)
		return err
	})
	return err
}

func (boltdb *BoltDB) Update(key []byte, fn func(bucket Bucket) error) error {
	err := boltdb.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(key)
		bucket := new(bucket)
		bucket.boltBucket = b
		err := fn(bucket)
		return err
	})
	return err
}

func (boltdb *BoltDB) Create(key []byte) (Bucket, error) {
	tx, err := boltdb.DB.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	boltBucket, err := tx.CreateBucket(key)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	bucket := new(bucket)
	bucket.boltBucket = boltBucket
	return bucket, nil
}

func (boltdb *BoltDB) CreateIfNotExists(key []byte) (Bucket, error) {
	tx, err := boltdb.DB.Begin(true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	blotBucket, err := tx.CreateBucketIfNotExists(key)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	bucket := new(bucket)
	bucket.boltBucket = blotBucket
	return bucket, nil
}

func (boltdb *BoltDB) Delete(key []byte) error {
	tx, err := boltdb.DB.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.DeleteBucket(key)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
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
