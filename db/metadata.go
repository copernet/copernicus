package db

// MetaData represents a collection of Bucket
type MetaData interface {
	Create(key []byte) (Bucket, error)

	CreateIfNotExists(key []byte) (Bucket, error)

	Get(key []byte) Bucket

	Delete(key []byte) error
}
