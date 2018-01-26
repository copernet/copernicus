package db

type DB interface {
	// Type returns the database driver type the current database instance
	Type() string

	Begin(writable bool) (Transaction, error)

	View(fn func(tx Transaction) error) errorgit stat

	Update(fn func(tx Transaction) error) error

	Close() error
}
