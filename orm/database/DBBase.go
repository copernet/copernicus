package database

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
