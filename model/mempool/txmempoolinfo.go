package mempool

import (
	"encoding/binary"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"io"
)

type TxMempoolInfo struct {
	Tx      *tx.Tx       // The transaction itself
	Time    int64        // Time the transaction entered the memPool
	FeeRate util.FeeRate // FeeRate of the transaction
}

func (info *TxMempoolInfo) Serialize(w io.Writer) error {
	err := info.Tx.Serialize(w)
	if err != nil {
		return err
	}

	err = util.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(info.Time))
	if err != nil {
		return err
	}

	err = util.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(info.FeeRate.SataoshisPerK))
	return err
}

func (info *TxMempoolInfo) Deserialize(r io.Reader) error {
	err := info.Tx.Unserialize(r)
	if err != nil {
		return err
	}

	t, err := util.BinarySerializer.Uint64(r, binary.LittleEndian)
	if err != nil {
		return err
	}

	f, err := util.BinarySerializer.Uint64(r, binary.LittleEndian)
	if err != nil {
		return err
	}
	info.Time = int64(t)
	info.FeeRate.SataoshisPerK = int64(f)
	return nil
}
