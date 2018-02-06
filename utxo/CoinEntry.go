package utxo

import (
	"io"

	"bytes"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/utils"
)

type CoinEntry struct {
	outpoint *model.OutPoint
	key      byte
}

func (coinEntry *CoinEntry) Serialize(writer io.Writer) error {
	_, err := writer.Write([]byte{coinEntry.key})
	if err != nil {
		return err
	}
	err = utils.WriteVarBytes(writer, coinEntry.outpoint.Hash.GetCloneBytes())
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(writer, uint64(coinEntry.outpoint.Index))
	return nil

}

func DeserializeCE(reader io.Reader) (coinEntry *CoinEntry, err error) {
	coinEntry = new(CoinEntry)
	coinEntry.outpoint.Hash = utils.Hash{}
	keys := make([]byte, 1)
	_, err = io.ReadFull(reader, keys)
	if err != nil {
		return
	}

	bytes, err := utils.ReadVarBytes(reader, 32, "hash")
	if err != nil {
		return
	}
	n, err := utils.ReadVarInt(reader)
	if err != nil {
		return
	}
	coinEntry.key = keys[0]
	coinEntry.outpoint.Hash.SetBytes(bytes)
	coinEntry.outpoint.Index = uint32(n)
	return
}

func (coinEntry *CoinEntry) GetSerKey() []byte {
	buf := bytes.NewBuffer(nil)
	coinEntry.Serialize(buf)
	return buf.Bytes()
}

func NewCoinEntry(outPoint *model.OutPoint) *CoinEntry {
	coinEntry := new(CoinEntry)
	coinEntry.outpoint = outPoint
	coinEntry.key = orm.DB_COIN
	return coinEntry
}
