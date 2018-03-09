package boltdb

type DBBase interface {
	// Type returns the database driver type the current database instance
	Type() string

	View(key []byte, fn func(bucket Bucket) error) error

	Update(key []byte, fn func(bucket Bucket) error) error

	Create(key []byte) (Bucket, error)

	CreateIfNotExists(key []byte) (Bucket, error)

	Delete(key []byte) error

	Close() error
}

type Bucket interface {
	ForEach(func(k, v []byte) error) error

	Cursor() Cursor

	Writable() bool

	Put(key, value []byte) error

	Get(key []byte) []byte

	Delete(key []byte) error

	Exists(key []byte) bool

	EstimateSize() int
}

// Cursor represents a cursor over key/value pairs
type Cursor interface {
	Bucket() Bucket
	// Delete removes the current key/value pair the cursor is at without
	// invalidating the cursor.
	Delete() error
	// First positions the cursor at the first key/value pair and returns
	// whether or not the pair exists.
	First() bool
	// Last positions the cursor at the last key/value pair and returns
	// whether or not the pair exists.
	Last() bool
	// Next moves the cursor one key/value pair forward and returns whether
	// or not the pair exists.
	Next() bool
	// Prev moves the cursor one key/value pair backward and returns whether
	// or not the pair exists.
	Prev() bool
	// Seek positions the cursor at the first key/value pair that is greater
	// than or equal to the passed seek key.  Returns whether or not the
	// pair exists.
	Seek(seek []byte) bool
	// Key returns the current key the cursor is pointing to.
	Key() []byte
	// Value returns the current value the cursor is pointing to.
	Value() []byte
}
