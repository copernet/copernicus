package database

type Bucket interface {
	ForEach(func(k, v []byte) error) error

	Cursor() Cursor

	Writable() bool

	Put(key, value []byte) error

	Get(key []byte) []byte

	Delete(key []byte) error
}
