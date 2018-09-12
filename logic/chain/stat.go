package chain

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/copernet/copernicus/conf"
	mchain "github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/outpoint"
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
	return fmt.Sprintf("height=%d,bestblock=%s,transactions=%d,txouts=%d,"+
		"hash_serialized=%s,total_amount=%d\n",
		s.height, s.bestblock.String(), s.nTx, s.nTxOuts, s.hashSerialized.String(), s.amount)
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
	if err := util.WriteVarLenInt(hashbuf, uint64(0)); err != nil {
		return err
	}
	return nil
}

func GetUTXOStats(cdb utxo.CoinsDB, stat *stat) error {
	besthash, err := cdb.GetBestBlock()
	if err != nil {
		fmt.Printf("GetBestBlock(), failed=%v\n", err)
		return err
	}
	stat.bestblock = *besthash
	stat.height = int(mchain.GetInstance().FindBlockIndex(*besthash).Height)

	hashbuf := bytes.NewBuffer(nil)
	hashbuf.Write(stat.bestblock[:])
	prevkey := util.Hash{}
	outpoint := outpoint.OutPoint{}
	bw := bytes.NewBuffer(nil)
	outputs := make(map[uint32]*utxo.Coin)

	iter := cdb.GetDBW().Iterator(nil)
	defer iter.Close()
	iter.Seek([]byte{db.DbCoin})

	logf, err := os.OpenFile(filepath.Join(conf.DataDir, "coins.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer logf.Close()
	//fmt.Fprintf(logf, "besthash=%s\n", stat.bestblock.String())
	for ; iter.Valid(); iter.Next() {
		bw.Reset()
		bw.Write(iter.GetKey())
		entry := utxo.NewCoinKey(&outpoint)
		if err := entry.Unserialize(bw); err != nil {
			return err
		}
		bw.Reset()
		bw.Write(iter.GetVal())
		coin := new(utxo.Coin)
		if err := coin.Unserialize(bw); err != nil {
			return err
		}
		//fmt.Fprintf(logf, "outpoint=(%s,%d)\n", outpoint.Hash.String(), outpoint.Index)
		//fmt.Fprintf(logf, "coin=%+v,script=%s\n", coin, hex.EncodeToString(coin.GetScriptPubKey().GetData()))
		if len(outputs) > 0 && outpoint.Hash != prevkey {
			applyStats(stat, hashbuf, &prevkey, outputs)
			for k := range outputs {
				delete(outputs, k)
			}
		}
		outputs[outpoint.Index] = coin
		prevkey = outpoint.Hash
	}
	if len(outputs) > 0 {
		applyStats(stat, hashbuf, &prevkey, outputs)
	}
	stat.hashSerialized = util.DoubleSha256Hash(hashbuf.Bytes())
	return nil
}
