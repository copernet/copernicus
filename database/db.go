package database

import (
	"crypto/rand"
	"os"

	rocks "github.com/tecbot/gorocksdb"
)

const (
	obfuscateKeyKey = "\000obfuscate_key"
	obfuscateKeyLen = 8
)

type DBWrapper struct {
	option       *rocks.Options
	readOption   *rocks.ReadOptions
	iterOption   *rocks.ReadOptions
	writeOption  *rocks.WriteOptions
	syncOption   *rocks.WriteOptions
	db           *rocks.DB
	obfuscateKey []byte
}

func genObfuscateKey() []byte {
	buf := make([]byte, obfuscateKeyLen)
	_, err := rand.Read(buf)
	if err != nil {
		panic("failed read random bytes")
	}
	return buf
}

func NewDBWrapper(filePath string, cacheSize int, wipe bool, obfuscate bool) (*DBWrapper, error) {
	opts := getOptions(cacheSize)
	if wipe {
		if err := rocks.DestroyDb(filePath, opts); err != nil {
			return nil, err
		}
	}
	err := os.MkdirAll(filePath, 0740)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	db, err := rocks.OpenDb(opts, filePath)
	if err != nil {
		return nil, err
	}

	ro := rocks.NewDefaultReadOptions()
	io := rocks.NewDefaultReadOptions()
	wo := rocks.NewDefaultWriteOptions()
	so := rocks.NewDefaultWriteOptions()

	ro.SetVerifyChecksums(true)
	io.SetVerifyChecksums(true)
	io.SetFillCache(false)
	so.SetSync(true)

	dbw := &DBWrapper{
		option:      opts,
		readOption:  ro,
		iterOption:  io,
		writeOption: wo,
		syncOption:  so,
		db:          db,
	}
	exists := dbw.Exists([]byte(obfuscateKeyKey))
	if !exists && obfuscate && dbw.IsEmpty() {
		newKey := genObfuscateKey()
		if err = dbw.Write([]byte(obfuscateKeyKey), newKey, false); err != nil {
			return nil, err
		}
		dbw.obfuscateKey = newKey
	}
	return dbw, nil
}

func xor(val, key []byte) {
	if len(key) == 0 {
		return
	}
	for i, j := 0, 0; i < len(val); i, j = i+1, j+1 {
		val[i] ^= key[j]
		if j == len(key) {
			j = 0
		}
	}
}

func (dbw *DBWrapper) Read(key []byte) ([]byte, error) {
	value, err := dbw.db.Get(dbw.readOption, key)
	if err != nil {
		return nil, err
	}
	defer value.Free()
	var val []byte
	val = append(val, value.Data()...)

	xor(val, dbw.obfuscateKey)
	return val, nil
}

func (dbw *DBWrapper) Write(key, val []byte, sync bool) error {
	bw := NewBatchWrapper(dbw)
	defer bw.Close()
	bw.Put(key, val)
	return dbw.WriteBatch(bw, sync)
}

func (dbw *DBWrapper) WriteBatch(bw *BatchWrapper, sync bool) error {
	var opts *rocks.WriteOptions
	if sync {
		opts = dbw.syncOption
	} else {
		opts = dbw.writeOption
	}
	return dbw.db.Write(opts, bw.batch)
}

func (dbw *DBWrapper) Exists(key []byte) bool {
	slice, err := dbw.db.Get(dbw.readOption, key)
	if err != nil {
		return false
	}
	defer slice.Free()
	return slice.Size() > 0
}

func (dbw *DBWrapper) Erase(key []byte, sync bool) error {
	bw := NewBatchWrapper(dbw)
	defer bw.Close()
	bw.Erase(key)
	return dbw.WriteBatch(bw, sync)
}

func (dbw *DBWrapper) Sync() error {
	bw := NewBatchWrapper(dbw)
	defer bw.Close()
	return dbw.WriteBatch(bw, true)
}

func (dbw *DBWrapper) NewIterator() *IterWrapper {
	return NewIterWrapper(dbw, dbw.db.NewIterator(dbw.iterOption))
}

func (dbw *DBWrapper) IsEmpty() bool {
	it := dbw.NewIterator()
	it.SeekToFirst()
	return !it.Valid()
}

func (dbw *DBWrapper) EstimateSize(begin, end []byte) uint64 {
	rs := []rocks.Range{{Start: begin, Limit: end}}
	return dbw.db.GetApproximateSizes(rs)[0]
}

func (dbw *DBWrapper) GetObfuscateKey() []byte {
	return dbw.obfuscateKey
}

func (dbw *DBWrapper) Close() {
	if dbw.option != nil {
		dbw.option.Destroy()
	}
	if dbw.readOption != nil {
		dbw.readOption.Destroy()
	}
	if dbw.writeOption != nil {
		dbw.writeOption.Destroy()
	}
	if dbw.iterOption != nil {
		dbw.iterOption.Destroy()
	}
	if dbw.syncOption != nil {
		dbw.syncOption.Destroy()
	}
	if dbw.db != nil {
		dbw.db.Close()
	}
}

func getOptions(cacheSize int) *rocks.Options {
	bbto := rocks.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(rocks.NewLRUCache(cacheSize / 2))
	bbto.SetFilterPolicy(rocks.NewBloomFilter(10))

	opts := rocks.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCompression(rocks.NoCompression)
	opts.SetWriteBufferSize(cacheSize / 4)
	opts.SetMaxOpenFiles(64)
	opts.SetCreateIfMissing(true)
	return opts
}

type BatchWrapper struct {
	parent  *DBWrapper
	batch   *rocks.WriteBatch
	sizeEst int
}

func NewBatchWrapper(parent *DBWrapper) *BatchWrapper {
	return &BatchWrapper{
		parent: parent,
		batch:  rocks.NewWriteBatch(),
	}
}

func (bw *BatchWrapper) Close() {
	if bw.batch != nil {
		bw.batch.Destroy()
	}
}

func (bw *BatchWrapper) Clear() {
	bw.batch.Clear()
	bw.sizeEst = 0
}

func (bw *BatchWrapper) Put(key, val []byte) {
	xor(val, bw.parent.GetObfuscateKey())
	bw.batch.Put(key, val)
}

func (bw *BatchWrapper) SizeEstimate() int {
	return bw.sizeEst
}

func (bw *BatchWrapper) Erase(key []byte) {
	bw.batch.Delete(key)
}

type IterWrapper struct {
	parent *DBWrapper
	iter   *rocks.Iterator
}

func NewIterWrapper(parent *DBWrapper, iter *rocks.Iterator) *IterWrapper {
	return &IterWrapper{
		parent: parent,
		iter:   iter,
	}
}

func (iw *IterWrapper) Valid() bool {
	if iw.iter == nil {
		return false
	}
	return iw.iter.Valid()
}

func (iw *IterWrapper) SeekToFirst() {
	if iw.iter != nil {
		iw.iter.SeekToFirst()
	}
}

func (iw *IterWrapper) GetKey() []byte {
	var key []byte
	if iw.iter != nil {
		s := iw.iter.Key()
		key = append(key, s.Data()...)
		s.Free()
	}
	return key
}

func (iw *IterWrapper) GetKeySize() int {
	size := 0
	if iw.iter != nil {
		size = iw.iter.Key().Size()
	}
	return size
}

func (iw *IterWrapper) GetVal() []byte {
	var val []byte
	if iw.iter != nil {
		s := iw.iter.Value()
		val = append(val, s.Data()...)
		s.Free()

	}
	return val
}

func (iw *IterWrapper) GetValSize() int {
	size := 0
	if iw.iter != nil {
		size = iw.iter.Value().Size()
	}
	return size
}

func (iw *IterWrapper) Seek(key []byte) {
	if iw.iter != nil {
		iw.iter.Seek(key)
	}
}
func (iw *IterWrapper) Next() {
	if iw.iter != nil {
		iw.iter.Next()
	}
}

func (iw *IterWrapper) Close() {
	if iw.iter != nil {
		iw.iter.Close()
	}
}
