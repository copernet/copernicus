package orm

import (
	"crypto"
)

type DBTx interface {
	Metadata() MetaData

	HasBlock(hash *crypto.Hash) (bool, error)

	HasBlocks(hashes []crypto.Hash) ([]bool, error)

	FetchBlockHeader(hash *crypto.Hash) ([]byte, error)

	FetchBlockHeaders(hashes []crypto.Hash) ([][]byte, error)

	FetchBlock(hash *crypto.Hash) ([]byte, error)

	FetchBlocks(hashes []crypto.Hash) ([][]byte, error)

	Commit() error

	Rollback() error
}
