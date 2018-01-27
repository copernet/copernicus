package orm

type DBBase interface {
	// Type returns the database driver type the current database instance
	Type() string
	
	View(fn func(bucket Bucket) error) error
	
	Update(fn func(bucket Bucket) error) error
	
	Close() error
}
