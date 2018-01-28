package boltdb

import (
	"testing"
	"io/ioutil"
	"os"
	"github.com/boltdb/bolt"
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
