package utxo

import (
	"bytes"
	"errors"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/database"
)

var ErrCoinEntry = errors.New("nil CoinEntry")

type CoinEntry struct {
	outpoint core.OutPoint
	key      byte
}

func (ce *CoinEntry) Serialize(w io.Writer) error {
	if ce == nil {
		return ErrCoinEntry
	}
	_, err := w.Write([]byte{ce.key})
	if err != nil {
		return err
	}
	if _, err = w.Write(ce.outpoint.Hash[:]); err != nil {
		return err
	}
	err = utils.WriteVarLenInt(w, uint64(ce.outpoint.Index))
	return err
}

func (ce *CoinEntry) Unserialize(r io.Reader) error {
	if ce == nil {
		return ErrCoinEntry
	}
	tmp := make([]byte, 32)
	if _, err := r.Read(tmp[:1]); err != nil {
		return err
	}
	ce.key = tmp[0]
	if _, err := r.Read(tmp); err != nil {
		return err
	}
	copy(ce.outpoint.Hash[:], tmp)
	n, err := utils.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	ce.outpoint.Index = uint32(n)
	return nil
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
