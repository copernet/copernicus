package orm

type DBBase interface {
	// Type returns the database driver type the current database instance
	Type() string

	Begin(writable bool) (Bucket, error)

	View(fn func(bucket Bucket) error) error

	Update(fn func(bucket Bucket) error) error

	Close() error
}
