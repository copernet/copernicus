package boltdb

import (
	"github.com/boltdb/bolt"
	"github.com/btcboost/copernicus/orm/database"
)

type cursor struct {
	database.Cursor
	bucket     database.Bucket
	boltCursor *bolt.Cursor
}

func (cursor *cursor) Bucket() database.Bucket {
	return cursor.bucket
}

func (cursor *cursor) Delete() error {
	return cursor.Cursor.Delete()
}

func (cursor *cursor) First() bool {
	return cursor.Cursor.First()
}

func (cursor *cursor) Last() bool {
	return cursor.Cursor.Last()
}

func (cursor *cursor) Next() bool {
	return cursor.Cursor.Next()
}

func (cursor *cursor) Prev() bool {
	return cursor.Cursor.Prev()
}

func (cursor *cursor) Seek(seek []byte) bool {
	return cursor.Cursor.Seek(seek)
}

func (cursor *cursor) Key() []byte {
	return cursor.Cursor.Key()
}

func (cursor *cursor) Value() []byte {
	return cursor.Cursor.Value()
}
