package db

type DB interface {
	// Type returns the database driver type the current database instance
	Type() string
	
	Begin(writable bool) (DBTx, error)
	
	View(fn func(tx DBTx) error) error
	
	Update(fn func(tx DBTx) error) error
	
	Close() error
}
