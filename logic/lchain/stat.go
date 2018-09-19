package lchain

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/copernet/copernicus/log"
	mchain "github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

type stat struct {
	height         int
	bestblock      util.Hash
	nTx            uint64
	nTxOuts        uint64
	hashSerialized util.Hash
	amount         int64
}

func (s *stat) String() string {
	return fmt.Sprintf("height=%d,bestblock=%s,hash_serialized=%s\n",
		s.height, s.bestblock.String(), s.hashSerialized.String())
}

func applyStats(stat *stat, hashbuf *bytes.Buffer, txid *util.Hash, outputs map[uint32]*utxo.Coin) error {
	txIndexSort := []uint32{}
	for k := range outputs {
		txIndexSort = append(txIndexSort, k)
	}
	sort.Slice(txIndexSort, func(i, j int) bool {
		return txIndexSort[i] < txIndexSort[j]
	})

	hashbuf.Write((*txid)[:])
	cb := int32(0)
	if outputs[txIndexSort[0]].IsCoinBase() {
		cb = 1
	}
	height := outputs[txIndexSort[0]].GetHeight()
	if err := util.WriteVarLenInt(hashbuf, uint64(height*2+cb)); err != nil {
		return err
	}
	stat.nTx++
	for _, k := range txIndexSort {
		v := outputs[k]
		if err := util.WriteVarLenInt(hashbuf, uint64(k+1)); err != nil {
			return err
		}
		util.WriteVarBytes(hashbuf, v.GetScriptPubKey().GetData())
		if err := util.WriteVarLenInt(hashbuf, uint64(v.GetAmount())); err != nil {
			return err
		}
		stat.nTxOuts++
		stat.amount += int64(v.GetAmount())
	}
	err := util.WriteVarLenInt(hashbuf, uint64(0))
	return err
}

func GetUTXOStats(cdb utxo.CoinsDB, stat *stat) error {
	besthash, err := cdb.GetBestBlock()
	if err != nil {
		log.Debug("in GetUTXOStats, GetBestBlock(), failed=%v\n", err)
		return err
	}
	stat.bestblock = *besthash
	stat.height = int(mchain.GetInstance().FindBlockIndex(*besthash).Height)

	hashbuf := bytes.NewBuffer(nil)
	hashbuf.Write(stat.bestblock[:])

	iter := cdb.GetDBW().Iterator(nil)
	defer iter.Close()
	iter.Seek([]byte{db.DbCoin})

	for ; iter.Valid(); iter.Next() {
		hashbuf.Write(iter.GetKey())
		hashbuf.Write(iter.GetVal())
	}
	stat.hashSerialized = util.Sha256Hash(hashbuf.Bytes())
	return nil
}
