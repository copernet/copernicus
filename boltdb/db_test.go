package boltdb

import (
	"io/ioutil"
	"os"
	"testing"
	
	"github.com/boltdb/bolt"
	"github.com/btcboost/copernicus/orm"
)

func TestOpen(t *testing.T) {
	path, err := TempFile("db-")
	if err != nil {
		t.Error(err)
	}
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		t.Error(err)
	} else if db == nil {
		t.Error("db is nil")
	}
	if s := db.Path(); s != path {
		t.Errorf("db path(%s) is not path(%s)", db.Path(), s)
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
	path, err := TempFile("db-bucket-")
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
	path, err := TempFile("db-bucket-")
	if err != nil {
		t.Error(err)
	}
	bolddb, err := NewBlotDB(path)
	_, err = bolddb.CreateIfNotExists([]byte(bucketKey))
	if err != nil {
		t.Error(err)
	}
	key := "key1"
	value := "value1"
	
	err = bolddb.Update([]byte(bucketKey), func(bucket orm.Bucket) error {
		err := bucket.Put([]byte(key), []byte(value))
		return err
	})
	if err != nil {
		t.Error(err)
	}
	bolddb.View([]byte(bucketKey), func(bucket orm.Bucket) error {
		v := bucket.Get([]byte(key))
		if string(v) != value {
			t.Errorf("get v(%s) from db is wrong", string(v))
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
