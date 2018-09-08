package utxo

import (
	"bytes"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/syndtr/goleveldb/leveldb"
)

type CoinsDB struct {
	dbw *db.Wrapper
}

func (coinsViewDB *CoinsDB) GetCoin(outpoint *outpoint.OutPoint) (*Coin, error) {
	buf := bytes.NewBuffer(nil)
	err := NewCoinKey(outpoint).Serialize(buf)
	if err != nil {
		log.Emergency("db.GetCoin err:%#v", err)
		panic("get coin is failed!")
	}
	log.Debug("outpoint==========", outpoint)
	coinBuff, err := coinsViewDB.dbw.Read(buf.Bytes())
	if err != nil {

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
		log.Emergency("db.HaveCoin err:%#v", err)

		return false
	}
	return coinsViewDB.dbw.Exists(buf.Bytes())
}

func (coinsViewDB *CoinsDB) GetBestBlock() (*util.Hash, error) {
	v, err := coinsViewDB.dbw.Read([]byte{db.DbBestBlock})
	if err == leveldb.ErrNotFound {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	hashBlock := new(util.Hash)
	if v == nil {
		return hashBlock, nil
	}
	_, err = hashBlock.Unserialize(bytes.NewBuffer(v))
	return hashBlock, err
}

func (coinsViewDB *CoinsDB) BatchWrite(cm map[outpoint.OutPoint]*Coin, hashBlock util.Hash) error {
	mapCoins := cm
	batch := db.NewBatchWrapper(coinsViewDB.dbw)
	count := 0
	changed := 0
	for k, v := range mapCoins {
		if v.dirty {
			entry := NewCoinKey(&k)
			bufEntry := bytes.NewBuffer(nil)
			err := entry.Serialize(bufEntry)
			if err != nil {
				log.Error("coinDB:serialize bufEntry failed %v", err)
			}

			if v.IsSpent() {
				batch.Erase(bufEntry.Bytes())
			} else {
				coinByte := bytes.NewBuffer(nil)
				err := v.Serialize(coinByte)
				if err != nil {
					log.Error("coinDB:serialize coinByte failed %v", err)
				}
				batch.Write(bufEntry.Bytes(), coinByte.Bytes())
			}
			changed++
		}
		count++
		delete(cm, k)
	}
	if !hashBlock.IsNull() {
		hashByte := bytes.NewBuffer(nil)
		_, err := hashBlock.Serialize(hashByte)
		if err != nil {
			log.Error("coinDB:Serialize hash block failed<%v>, please check.", err)
		}
		batch.Write([]byte{db.DbBestBlock}, hashByte.Bytes())
	}

	ret := coinsViewDB.dbw.WriteBatch(batch, false)

	return ret
}

func (coinsViewDB *CoinsDB) EstimateSize() uint64 {
	return coinsViewDB.dbw.EstimateSize([]byte{db.DbCoin}, []byte{db.DbCoin + 1})
}

func newCoinsDB(do *db.DBOption) *CoinsDB {
	if do == nil {
		return nil
	}

	dbw, err := db.NewDBWrapper(do)

	if err != nil {
		panic("init CoinsDB failed...")
	}

	return &CoinsDB{
		dbw: dbw,
	}
}
