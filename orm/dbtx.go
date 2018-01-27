package orm

import (
	"crypto"
)

type Transaction interface {
	// Metadata returns the top-most bucket for all metadata storage.
	Metadata() MetaData

	//	StoreBlock(block *model.Block) error

	HasBlock(hash *crypto.Hash) (bool, error)

	HasBlocks(hashes []crypto.Hash) ([]bool, error)

	FetchBlockHeader(hash *crypto.Hash) ([]byte, error)

	FetchBlockHeaders(hashes []crypto.Hash) ([][]byte, error)

	FetchBlock(hash *crypto.Hash) ([]byte, error)

	FetchBlocks(hashes []crypto.Hash) ([][]byte, error)

	Commit() error

	Rollback() error
}
