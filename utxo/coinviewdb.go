package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/utils"
)

type CoinsViewDB struct {
	dbw *database.DBWrapper
}

func (coinsViewDB *CoinsViewDB) GetCoin(outpoint *core.OutPoint) (*Coin, error) {
	buf := bytes.NewBuffer(nil)
	err := NewCoinEntry(outpoint).Serialize(buf)
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

func (coinsViewDB *CoinsViewDB) HaveCoin(outpoint *core.OutPoint) bool {
	buf := bytes.NewBuffer(nil)
	err := NewCoinEntry(outpoint).Serialize(buf)
	if err != nil {
		return false
	}
	return coinsViewDB.dbw.Exists(buf.Bytes())
}

func (coinViewDB *CoinViewDB) SetBestBlock(hash *utils.Hash) {
	var cvc *CoinsViewCache
	cvc.hashBlock = *hash
}

func (coinsViewDB *CoinsViewDB) GetBestBlock() utils.Hash {
	var hashBestChain utils.Hash
	buf := bytes.NewBuffer(nil)
	hashBestChain.Serialize(buf)
	v, err := coinsViewDB.dbw.Read([]byte{database.DbBestBlock})
	v = append(v, buf.Bytes()...)
	if err != nil {
		return utils.Hash{}
	}
	return hashBestChain
}

func (coinsViewDB *CoinsViewDB) BatchWrite(mapCoins *CacheCoins, hashBlock *utils.Hash) error {
	var batch *database.BatchWrapper
	count := 0
	changed := 0
	for k, v := range *mapCoins {
		if v.dirty {
			entry := NewCoinEntry(&k)
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
		batch.Write([]byte{database.DbBestBlock}, hashByte.Bytes())
	}

	ret := coinsViewDB.dbw.WriteBatch(batch, false)
	log.Print("coindb", "debug", "Committed %u changed transaction outputs (out of %u) to coin database...\n", changed, count)
	return ret
}

func (coinsViewDB *CoinsViewDB) EstimateSize() uint64 {
	return coinsViewDB.dbw.EstimateSize([]byte{database.DbCoin}, []byte{database.DbCoin + 1})
}

//func (coinsViewDB *CoinsViewDB) Cursor() *CoinsViewCursor {
//
//	// It seems that there are no "const iterators" for LevelDB. Since we only
//	// need read operations on it, use a const-cast to get around that
//	// restriction.
//
//}

func NewCoinsViewDB(do *database.DBOption) *CoinsViewDB {
	if do == nil {
		return nil
	}

	dbw, err := database.NewDBWrapper(&database.DBOption{
		FilePath:      conf.GetDataPath() + "/chainstate",
		CacheSize:     do.CacheSize,
		Wipe:          false,
		DontObfuscate: false,
	})

	if err != nil {
		panic("init CoinsViewDB failed...")
	}

	return &CoinsViewDB{
		dbw: dbw,
	}
}
