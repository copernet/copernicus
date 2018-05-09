package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/conf"

	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/util"

	"github.com/btcboost/copernicus/model/outpoint"
)

type CoinsDB struct {
	dbw *db.DBWrapper
}

func (coinsViewDB *CoinsDB) GetCoin(outpoint *outpoint.OutPoint) (*Coin, error) {
	buf := bytes.NewBuffer(nil)
	err := NewCoinKey(outpoint).Serialize(buf)
	if err != nil {
		panic("get coin is failed!")
	}

	coinBuff, err := coinsViewDB.dbw.Read(buf.Bytes())
	if err != nil{
		return nil, err
	}
	coin := NewEmptyCoin()
	err = coin.Unserialize(bytes.NewBuffer(coinBuff))
	return coin, err
}

func (coinsViewDB *CoinsDB) HaveCoin(outpoint *outpoint.OutPoint) bool {
	buf := bytes.NewBuffer(nil)
	err := NewCoinKey(outpoint).Serialize(buf)
	if err != nil {
		return false
	}
	return coinsViewDB.dbw.Exists(buf.Bytes())
}


func (coinsViewDB *CoinsDB) GetBestBlock() util.Hash {
	var hashBestChain util.Hash
	buf := bytes.NewBuffer(nil)
	hashBestChain.Serialize(buf)
	v, err := coinsViewDB.dbw.Read([]byte{db.DbBestBlock})
	v = append(v, buf.Bytes()...)
	if err != nil {
		return util.Hash{}
	}
	return hashBestChain
}

func (coinsViewDB *CoinsDB) BatchWrite(mapCoins *CoinsCacheMap, hashBlock *util.Hash) error {
	var batch *db.BatchWrapper
	count := 0
	changed := 0
	for k, v := range *mapCoins {
		if v.dirty {
			entry := NewCoinKey(&k)
			bufEntry := bytes.NewBuffer(nil)
			entry.Serialize(bufEntry)

			if v.Coin.IsSpent() {
				batch.Erase(bufEntry.Bytes())
			} else {
				coinByte := bytes.NewBuffer(nil)
				v.Coin.Serialize(coinByte)
				batch.Write(bufEntry.Bytes(), coinByte.Bytes())
			}
			changed++
		}
		count++
		delete(*mapCoins, k)
	}
	if !hashBlock.IsNull() {
		hashByte := bytes.NewBuffer(nil)
		hashBlock.Serialize(hashByte)
		batch.Write([]byte{db.DbBestBlock}, hashByte.Bytes())
	}

	ret := coinsViewDB.dbw.WriteBatch(batch, false)
	log.Print("coindb", "debug", "Committed %u changed transaction outputs (out of %u) to coin db...\n", changed, count)
	return ret
}

func (coinsViewDB *CoinsDB) EstimateSize() uint64 {
	return coinsViewDB.dbw.EstimateSize([]byte{db.DbCoin}, []byte{db.DbCoin + 1})
}

//func (coinsViewDB *CoinsDB) Cursor() *CoinsViewCursor {
//
//	// It seems that there are no "const iterators" for LevelDB. Since we only
//	// need read operations on it, use a const-cast to get around that
//	// restriction.
//
//}

func NewCoinsDB(do *db.DBOption) *CoinsDB {
	if do == nil {
		return nil
	}

	dbw, err := db.NewDBWrapper(&db.DBOption{
		FilePath:      conf.GetDataPath() + "/chainstate",
		CacheSize:     do.CacheSize,
		Wipe:          false,
		DontObfuscate: true,
	})

	if err != nil {
		panic("init CoinsDB failed...")
	}

	return &CoinsDB{
		dbw: dbw,
	}
}
