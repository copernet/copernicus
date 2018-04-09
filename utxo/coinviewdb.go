package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/utils"
)

type CoinViewDB struct {
	dbw *database.DBWrapper
}

func (coinViewDB *CoinViewDB) GetCoin(outpoint *core.OutPoint) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewCoinEntry(outpoint).Serialize(buf)
	if err != nil {
		panic("get coin is failed!")
	}

	return coinViewDB.dbw.Read(buf.Bytes())
}

func (coinViewDB *CoinViewDB) HaveCoin(outpoint *core.OutPoint) bool {
	buf := bytes.NewBuffer(nil)
	err := NewCoinEntry(outpoint).Serialize(buf)
	if err != nil {
		return false
	}
	return coinViewDB.dbw.Exists(buf.Bytes())
}

func (coinViewDB *CoinViewDB) SetBestBlock(hash *utils.Hash) {
	var cvc *CoinsViewCache
	cvc.hashBlock = *hash
}

func (coinViewDB *CoinViewDB) GetBestBlock() utils.Hash {
	var hashBestChain utils.Hash
	buf := bytes.NewBuffer(nil)
	hashBestChain.Serialize(buf)
	v, err := coinViewDB.dbw.Read([]byte{DbBestBlock})
	v = append(v, buf.Bytes()...)
	if err != nil {
		return utils.Hash{}
	}
	return hashBestChain
}

func (coinViewDB *CoinViewDB) BatchWrite(mapCoins map[core.OutPoint]CoinsCacheEntry) (bool, error) {
	return true, nil
}

func (coinViewDB *CoinViewDB) EstimateSize() uint64 {
	return coinViewDB.dbw.EstimateSize([]byte{DbCoin}, []byte{DbCoin + 1})
}

//func (coinViewDB *CoinViewDB) Cursor() *CoinsViewCursor {
//
//	// It seems that there are no "const iterators" for LevelDB. Since we only
//	// need read operations on it, use a const-cast to get around that
//	// restriction.
//
//}

func NewCoinViewDB(do *database.DBOption) *CoinViewDB {
	if do == nil {
		return nil
	}

	dbw, err := database.NewDBWrapper(&database.DBOption{
		FilePath:      conf.GetDataPath() + "/chainstate",
		CacheSize:     do.CacheSize,
		Wipe:          false,
		DontObfuscate: true,
	})

	if err != nil {
		panic("init CoinViewDB failed...")
	}

	return &CoinViewDB{
		dbw: dbw,
	}
}
