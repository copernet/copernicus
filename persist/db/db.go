package db

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"

	lvldb "github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	obfuscateKeyKey = "\000obfuscate_key"
	obfuscateKeyLen = 8
)

const (
	DbCoin       byte = 'C'
	DbCoins      byte = 'c'
	DbBlockFiles byte = 'f'
	DbTxIndex    byte = 't'
	DbBlockIndex byte = 'b'

	DbBestBlock   byte = 'B'
	DbFlag        byte = 'F'
	DbReindexFlag byte = 'R'
	DbLastBlock   byte = 'l'
	DbMaxBlock    byte = 'm'
)

const (
	preallocKeySize   = 64
	preallocValueSize = 1024
)

type DBWrapper struct {
	option       opt.Options
	readOption   opt.ReadOptions
	iterOption   opt.ReadOptions
	writeOption  opt.WriteOptions
	syncOption   opt.WriteOptions
	db           *lvldb.DB
	name         string
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

func getOptions(cacheSize int) opt.Options {
	var opts opt.Options
	opts.BlockCacher = opt.LRUCacher
	opts.BlockCacheCapacity = cacheSize / 2
	opts.WriteBuffer = cacheSize / 4
	opts.Filter = filter.NewBloomFilter(10)
	opts.Compression = opt.NoCompression
	opts.OpenFilesCacheCapacity = 64

	return opts
}

func destroyDB(path string) error {
	st, err := storage.OpenFile(path, false)
	if err != nil {
		return err
	}
	defer st.Close()
	fds, err := st.List(storage.TypeAll)
	if err != nil {
		return err
	}
	for _, fd := range fds {
		if err := st.Remove(fd); err != nil {
			return err
		}
	}
	for _, other := range []string{"CURRENT", "LOCK", "LOG", "LOG.old"} {
		if err := os.Remove(filepath.Join(path, other)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

type DBOption struct {
	FilePath       string
	CacheSize      int
	Wipe           bool
	DontObfuscate  bool
	ForceCompactdb bool
}

func NewDBWrapper(do *DBOption) (*DBWrapper, error) {
	if do == nil {
		return nil, errors.New("DBWrapper: nil DBOption")
	}
	opts := getOptions(do.CacheSize)
	if do.Wipe {
		if err := destroyDB(do.FilePath); err != nil {
			return nil, err
		}
	}

	err := os.MkdirAll(do.FilePath, 0740)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	db, err := lvldb.OpenFile(do.FilePath, &opts)
	if err != nil {
		return nil, err
	}
	if do.ForceCompactdb {
		if err := db.CompactRange(util.Range{}); err != nil {
			return nil, err
		}
	}

	ro := opt.ReadOptions{
		DontFillCache: false,
		Strict:        opt.StrictJournalChecksum | opt.StrictBlockChecksum,
	}
	io := opt.ReadOptions{
		DontFillCache: true,
		Strict:        opt.StrictJournalChecksum | opt.StrictBlockChecksum,
	}
	wo := opt.WriteOptions{}
	so := opt.WriteOptions{
		Sync: true,
	}

	dbw := &DBWrapper{
		option:      opts,
		readOption:  ro,
		iterOption:  io,
		writeOption: wo,
		syncOption:  so,
		db:          db,
		name:        filepath.Base(do.FilePath),
		//obfuscateKey: make([]byte, 8),
	}
	exists := false
	obk, err := dbw.Read([]byte(obfuscateKeyKey))
	if err == nil {
		dbw.obfuscateKey = obk
		exists = true
	}
	if !exists && !do.DontObfuscate && dbw.IsEmpty() {
		newKey := genObfuscateKey()
		if err := dbw.Write([]byte(obfuscateKeyKey), newKey, false); err != nil {
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
	for i, j := 0, 0; i < len(val); i++ {
		val[i] ^= key[j]
		j++
		if j == len(key) {
			j = 0
		}
	}
}

func (dbw *DBWrapper) Read(key []byte) ([]byte, error) {
	value, err := dbw.db.Get(key, &dbw.readOption)
	if err != nil {
		return nil, err
	}
	xor(value, dbw.obfuscateKey)
	return value, nil
}

func (dbw *DBWrapper) Write(key, val []byte, sync bool) error {
	bw := NewBatchWrapper(dbw)
	bw.Write(key, val)
	return dbw.WriteBatch(bw, sync)
}

func (dbw *DBWrapper) WriteBatch(bw *BatchWrapper, sync bool) error {
	var opts opt.WriteOptions
	if sync {
		opts = dbw.syncOption
	} else {
		opts = dbw.writeOption
	}
	return dbw.db.Write(&bw.bat, &opts)
}

func (dbw *DBWrapper) Exists(key []byte) bool {
	_, err := dbw.db.Get(key, &dbw.readOption)
	if err != nil {
		if err == lvldb.ErrNotFound {
			return false
		}
		panic("DBWrapper :" + err.Error())

	}
	return true
}

func (dbw *DBWrapper) Erase(key []byte, sync bool) error {
	bw := NewBatchWrapper(dbw)
	bw.Erase(key)
	return dbw.WriteBatch(bw, sync)
}

func (dbw *DBWrapper) Sync() error {
	bw := NewBatchWrapper(dbw)
	return dbw.WriteBatch(bw, true)
}

func (dbw *DBWrapper) Iterator() *IterWrapper {
	return NewIterWrapper(dbw, dbw.db.NewIterator(nil, &dbw.iterOption))
}

func (dbw *DBWrapper) IsEmpty() bool {
	it := dbw.Iterator()
	it.SeekToFirst()
	return !it.Valid()
}

func (dbw *DBWrapper) EstimateSize(begin, end []byte) uint64 {
	r := []util.Range{{Start: begin, Limit: end}}
	sizes, err := dbw.db.SizeOf(r)
	if err != nil {
		return 0
	}
	return uint64(sizes.Sum())
}

func (dbw *DBWrapper) CompactRange(begin, end []byte) error {
	return dbw.db.CompactRange(util.Range{Start: begin, Limit: end})
}

func (dbw *DBWrapper) GetObfuscateKey() []byte {
	return dbw.obfuscateKey
}

func (dbw *DBWrapper) Close() {
	if dbw.db != nil {
		dbw.db.Close()
	}
}

type BatchWrapper struct {
	bat     lvldb.Batch
	parent  *DBWrapper
	bkey    []byte
	bval    []byte
	sizeEst int
}

func NewBatchWrapper(parent *DBWrapper) *BatchWrapper {
	return &BatchWrapper{
		parent: parent,
		bkey:   make([]byte, 0, preallocKeySize),
		bval:   make([]byte, 0, preallocValueSize),
	}
}

func (bw *BatchWrapper) Clear() {
	bw.bat.Reset()
	bw.sizeEst = 0
}

func (bw *BatchWrapper) Write(key, val []byte) {
	bw.bkey = append(bw.bkey, key...)
	bw.bval = append(bw.bval, val...)
	//log.Printf("key,val:%s,%s\n", bw.bkey, (bw.bval))
	//log.Printf("bw.parent.GetObfuscateKey():%s\n", bw.parent.GetObfuscateKey())
	xor(bw.bval, bw.parent.GetObfuscateKey())
	bw.bat.Put(bw.bkey, bw.bval)
	// LevelDB serializes writes as:
	// - byte: header
	// - varint: key length (1 byte up to 127B, 2 bytes up to 16383B, ...)
	// - byte[]: key
	// - varint: value length
	// - byte[]: value
	// The formula below assumes the key and value are both less than 16k.
	k := 0
	v := 0
	if len(bw.bkey) > 127 {
		k = 1
	}
	if len(bw.bval) > 127 {
		v = 1
	}
	bw.sizeEst += 3 + k + len(bw.bkey) + v + len(bw.bval)
	bw.bkey = bw.bkey[:0]
	bw.bval = bw.bkey[:0]
}

func (bw *BatchWrapper) SizeEstimate() int {
	return bw.sizeEst
}

func (bw *BatchWrapper) Erase(key []byte) {
	bw.bkey = append(bw.bkey, key...)
	bw.bat.Delete(bw.bkey)
	k := 0
	if len(bw.bkey) > 127 {
		k = 1
	}
	bw.sizeEst += 2 + k + len(bw.bkey)
	bw.bkey = bw.bkey[:0]
}

type IterWrapper struct {
	parent *DBWrapper
	iter   iterator.Iterator
}

func NewIterWrapper(parent *DBWrapper, iter iterator.Iterator) *IterWrapper {
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
	iw.Seek(nil)
}

func (iw *IterWrapper) GetKey() []byte {
	var key []byte
	if iw.iter != nil {
		k := iw.iter.Key()
		key = append(key, k...)
	}
	return key
}

func (iw *IterWrapper) GetKeySize() int {
	return len(iw.GetKey())
}

func (iw *IterWrapper) GetVal() []byte {
	var val []byte
	if iw.iter != nil {
		v := iw.iter.Value()
		val = append(val, v...)
	}
	xor(val, iw.parent.GetObfuscateKey())
	return val
}

func (iw *IterWrapper) GetValSize() int {
	return len(iw.GetVal())
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
		iw.iter.Release()
	}
}
