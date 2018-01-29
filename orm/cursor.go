package orm

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
