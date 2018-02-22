package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/btcboost/copernicus/orm/database"
)

func TestOpen(t *testing.T) {
	path, err := TempFile("database-")
	if err != nil {
		t.Error(err)
	}
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		t.Error(err)
	} else if db == nil {
		t.Error("database is nil")
	}
	if s := db.Path(); s != path {
		t.Errorf("database path(%s) is not path(%s)", db.Path(), s)
	}
	if err := db.Close(); err != nil {
		t.Error(err)
	}
	if err := os.Remove(path); err != nil {
		t.Error(err)
	}
}

func TestCreateBucket(t *testing.T) {
	bucketKey := "test-bolt"
	path, err := TempFile("database-bucket-")
	if err != nil {
		t.Error(err)
	}
	boltdb, err := NewBlotDB(path)
	if err != nil {
		t.Error(err)
	}
	bucket, err := boltdb.Create([]byte(bucketKey))
	if err != nil {
		t.Error(err)
	}
	if !bucket.Writable() {
		t.Error("bucket is not writable")
	}
	if err := os.Remove(path); err != nil {
		t.Error(err)
	}

}

func TestPutKV(t *testing.T) {
	bucketKey := "test-bolt"
	path, err := TempFile("database-bucket-")
	if err != nil {
		t.Error(err)
	}
	boltdb, err := NewBlotDB(path)
	_, err = boltdb.CreateIfNotExists([]byte(bucketKey))
	if err != nil {
		t.Error(err)
	}
	key := "key1"
	value := "value1"

	err = boltdb.Update([]byte(bucketKey), func(bucket database.Bucket) error {
		err := bucket.Put([]byte(key), []byte(value))
		return err
	})
	if err != nil {
		t.Error(err)
	}
	err = boltdb.View([]byte(bucketKey), func(bucket database.Bucket) error {
		v := bucket.Get([]byte(key))
		if string(v) != value {
			t.Errorf("get v(%s) from database is wrong", string(v))
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}

}

func TestDeleteKV(t *testing.T) {
	bucketKey := "test-bolt"
	path, err := TempFile("database-bucket-")
	if err != nil {
		t.Error(err)
	}
	boltdb, err := NewBlotDB(path)
	_, err = boltdb.CreateIfNotExists([]byte(bucketKey))
	if err != nil {
		t.Error(err)
	}
	key := "key1"
	value := "value1"
	err = boltdb.Update([]byte(bucketKey), func(bucket database.Bucket) error {
		err := bucket.Put([]byte(key), []byte(value))
		return err
	})
	if err != nil {
		t.Error(err)
	}
	err = boltdb.View([]byte(bucketKey), func(bucket database.Bucket) error {
		v := bucket.Get([]byte(key))
		if v == nil {
			t.Error("put KV is wrong")
		}
		return nil
	})
	err = boltdb.Update([]byte(bucketKey), func(bucket database.Bucket) error {
		err := bucket.Delete([]byte(key))
		return err
	})
	if err != nil {
		t.Error(err)
	}

	err = boltdb.View([]byte(bucketKey), func(bucket database.Bucket) error {
		v := bucket.Get([]byte(key))
		if v != nil {
			t.Error("delete KV is wrong")
		}
		return nil
	})
}

func TempFile(prefix string) (string, error) {
	f, err := ioutil.TempFile("", prefix)
	if err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(f.Name()); err != nil {
		return "", err
	}
	return f.Name(), nil
}
