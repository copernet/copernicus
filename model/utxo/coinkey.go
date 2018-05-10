package utxo

import (
	"bytes"
	"io"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/persist/db"
)

type CoinKey struct {
	outpoint *outpoint.OutPoint
}

func (coinKey *CoinKey) Serialize(writer io.Writer) error {
	_, err := writer.Write([]byte{db.DbCoin})
	if err != nil {
		return err
	}
	err = util.WriteVarBytes(writer, coinKey.outpoint.Hash.GetCloneBytes())
	if err != nil {
		return err
	}
	err = util.WriteVarInt(writer, uint64(coinKey.outpoint.Index))
	return nil

}

func (coinKey *CoinKey)Unserialize(reader io.Reader) error {
	coinKey.outpoint.Hash = util.Hash{}
	keys := make([]byte, 1)
	_, err := io.ReadFull(reader, keys)
	if err != nil {
		return err
	}
	b, err := util.ReadVarBytes(reader, 32, "hash")
	if err != nil {
		return err
	}
	n, err := util.ReadVarInt(reader)
	if err != nil {
		return err
	}
	coinKey.outpoint.Hash.SetBytes(b)
	coinKey.outpoint.Index = uint32(n)
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
