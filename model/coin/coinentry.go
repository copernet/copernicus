package coin

import (
	"bytes"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/util"
)

type CoinEntry struct {
	outpoint *core.OutPoint
	key      byte
}

func (coinEntry *CoinEntry) Serialize(writer io.Writer) error {
	_, err := writer.Write([]byte{coinEntry.key})
	if err != nil {
		return err
	}
	err = util.WriteVarBytes(writer, coinEntry.outpoint.Hash.GetCloneBytes())
	if err != nil {
		return err
	}
	err = util.WriteVarInt(writer, uint64(coinEntry.outpoint.Index))
	return nil

}

func UnserializeCE(reader io.Reader) (coinEntry *CoinEntry, err error) {
	coinEntry = new(CoinEntry)
	coinEntry.outpoint.Hash = util.Hash{}
	keys := make([]byte, 1)
	_, err = io.ReadFull(reader, keys)
	if err != nil {
		return
	}

	b, err := util.ReadVarBytes(reader, 32, "hash")
	if err != nil {
		return
	}
	n, err := util.ReadVarInt(reader)
	if err != nil {
		return
	}
	coinEntry.key = keys[0]
	coinEntry.outpoint.Hash.SetBytes(b)
	coinEntry.outpoint.Index = uint32(n)
	return
}

func (coinEntry *CoinEntry) GetSerKey() []byte {
	buf := bytes.NewBuffer(nil)
	coinEntry.Serialize(buf)
	return buf.Bytes()
}

func NewCoinEntry(outPoint *core.OutPoint) *CoinEntry {
	coinEntry := new(CoinEntry)
	coinEntry.outpoint = outPoint
	coinEntry.key = database.DbCoin
	return coinEntry
}
