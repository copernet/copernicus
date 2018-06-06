package utxo

import (
	"bytes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"io"
)

type CoinKey struct {
	outpoint *outpoint.OutPoint
}

func (coinKey *CoinKey) Serialize(writer io.Writer) error {
	_, err := writer.Write([]byte{db.DbCoin})
	if err != nil {
		return err
	}
	err = coinKey.outpoint.Serialize(writer)
	return err

}

func (coinKey *CoinKey) Unserialize(reader io.Reader) error {
	coinKey.outpoint.Hash = util.Hash{}
	keys := make([]byte, 1)
	_, err := io.ReadFull(reader, keys)
	if err != nil {
		return err
	}
	err = coinKey.outpoint.Unserialize(reader)
	return err
}

func (coinKey *CoinKey) GetSerKey() []byte {
	buf := bytes.NewBuffer(nil)
	coinKey.Serialize(buf)
	return buf.Bytes()
}

func NewCoinKey(outPoint *outpoint.OutPoint) *CoinKey {
	coinKey := new(CoinKey)
	coinKey.outpoint = outPoint
	return coinKey
}
