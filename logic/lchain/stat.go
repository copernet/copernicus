package lchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"strconv"
)

var timeFile *os.File

type stat struct {
	height         int
	bestblock      util.Hash
	nTx            uint64
	nTxOuts        uint64
	hashSerialized util.Hash
	amount         int64
	bogoSize       uint64
}

type UTXOStat struct {
	Height         int
	BestBlock      util.Hash
	TxCount        uint64
	TxOutsCount    uint64
	HashSerialized util.Hash
	Amount         int64
	BogoSize       uint64
	DiskSize       uint64
}

func (s *stat) String() string {
	return fmt.Sprintf("height=%d,bestblock=%s,hash_serialized=%s\n",
		s.height, s.bestblock, s.hashSerialized)
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
		txOut := v.GetTxOut()
		stat.nTxOuts++
		stat.amount += int64(v.GetAmount())
		stat.bogoSize += 32 /* txid */ + 4 /* vout index */ + 4 /* height + coinbase */ +
			8 /* amount */ + 2 /* scriptPubKey len */ +
			uint64(txOut.GetScriptPubKey().SerializeSize()) /* scriptPubKey */
	}
	err := util.WriteVarLenInt(hashbuf, uint64(0))
	return err
}

func GetUTXOStats(cdb utxo.CoinsDB) (*UTXOStat, error) {
	stat := &stat{}
	b := time.Now()
	besthash, err := cdb.GetBestBlock()
	if err != nil {
		log.Debug("in GetUTXOStats, GetBestBlock(), failed=%v\n", err)
		return nil, err
	}
	stat.bestblock = *besthash
	stat.height = int(chain.GetInstance().FindBlockIndex(*besthash).Height)

	h := sha256.New()
	h.Write(stat.bestblock[:])

	iter := cdb.GetDBW().Iterator(nil)
	defer iter.Close()
	iter.Seek([]byte{db.DbCoin})

	var prevHash util.Hash
	outputs := make(map[uint32]*utxo.Coin)
	for ; iter.Valid(); iter.Next() {
		outPoint := &outpoint.OutPoint{}
		if err = outPoint.Unserialize(bytes.NewBuffer(iter.GetKey())); err != nil {
			return nil, err
		}
		coin := utxo.NewEmptyCoin()
		if err = coin.Unserialize(bytes.NewBuffer(iter.GetVal())); err != nil {
			return nil, err
		}
		if outPoint.Hash != prevHash && len(outputs) > 0 {
			hashBuf := bytes.NewBuffer(nil)
			if err = applyStats(stat, hashBuf, &prevHash, outputs); err != nil {
				return nil, err
			}
			h.Write(hashBuf.Bytes())
			outputs = make(map[uint32]*utxo.Coin)
		}
		prevHash = outPoint.Hash
		outputs[outPoint.Index] = coin
	}
	if len(outputs) > 0 {
		hashBuf := bytes.NewBuffer(nil)
		if err = applyStats(stat, hashBuf, &prevHash, outputs); err != nil {
			return nil, err
		}
		h.Write(hashBuf.Bytes())
	}
	copy(stat.hashSerialized[:], h.Sum(nil))

	utxoStat := &UTXOStat{
		Height:         stat.height,
		BestBlock:      stat.bestblock,
		TxCount:        stat.nTx,
		TxOutsCount:    stat.nTxOuts,
		HashSerialized: stat.hashSerialized,
		Amount:         stat.amount,
		BogoSize:       stat.bogoSize,
		DiskSize:       cdb.EstimateSize(),
	}

	log.Info("GetUTXOStats iter utxo db time: %s", time.Since(b).String())
	return utxoStat, nil
}

type utxoTaskArg struct {
	iter *db.IterWrapper
	stat *stat
}

type utxoTaskControl struct {
	utxoResult chan string
	utxoTask   chan utxoTaskArg
	done       chan struct{}
	numWorker  int
	numTask    int
	logOnce    sync.Once
	utxoOnce   sync.Once
}

var taskControl *utxoTaskControl

func init() {
	taskControl = newUtxoTaskControl(1, 16)
}

func newUtxoTaskControl(numTask, numWorker int) *utxoTaskControl {
	if numTask < 0 {
		numTask = 0
	}
	if numWorker < 0 {
		numWorker = 1
	}
	return &utxoTaskControl{
		utxoResult: make(chan string, numTask),
		utxoTask:   make(chan utxoTaskArg, numTask),
		numWorker:  numWorker,
		numTask:    numTask,
		done:       make(chan struct{}),
	}
}

func (tc *utxoTaskControl) PushUtxoTask(arg utxoTaskArg) {
	tc.utxoTask <- arg
}

func (tc *utxoTaskControl) StartLogTask() {
	tc.logOnce.Do(tc.startLogTask)
}

func (tc *utxoTaskControl) StartUtxoTask() {
	tc.utxoOnce.Do(tc.startUtxoTask)
}

func (tc *utxoTaskControl) Stop() {
	close(tc.done)
}

func (tc *utxoTaskControl) startLogTask() {
	f, err := os.OpenFile(filepath.Join(conf.DataDir, "logs/utxo.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		log.Debug("os.OpenFile() failed with : %s", err)
		return
	}
	go func() {
		defer f.Close()
		for {
			select {
			case <-tc.done:
				return
			case str := <-tc.utxoResult:
				if _, err := f.WriteString(str); err != nil {
					log.Debug("f.WriteString() failed with : %s", err)
					return
				}
			}
		}
	}()
}

func (tc *utxoTaskControl) startUtxoTask() {
	var err error
	timeFile, err = os.OpenFile(filepath.Join(conf.DataDir, "logs/utxo_stat.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		log.Debug("os.OpenFile() failed with : %s", err)
		return
	}
	for i := 0; i < tc.numWorker; i++ {
		go func() {
			for {
				select {
				case <-tc.done:
					return
				case arg := <-tc.utxoTask:
					//t1 := time.Now()
					utxoStat(arg.iter, arg.stat, tc.utxoResult)
					//statElasped := time.Since(t1)
					//if _, err := timeFile.WriteString("height: " + strconv.Itoa(arg.stat.height) + ", stat time: " + statElasped.String() + "\n"); err != nil {
					//	panic("write file failed")
					//}
				}
			}
		}()
	}
}

func utxoStat(iter *db.IterWrapper, stat *stat, res chan<- string) {
	defer iter.Close()

	tStatStart := time.Now()

	h := sha256.New()
	var key []byte
	var value []byte
	var tReadElasp time.Duration
	var tSha256Elasp time.Duration
	var tIterValidElasp time.Duration
	var tIterNextElasp time.Duration

	tSha256Start := time.Now()
	h.Write(stat.bestblock[:])
	tSha256Elasp += time.Since(tSha256Start)

	var dataSize int
	var i int
	//timeFile.WriteString(time.Now().String() + ", height: " + strconv.Itoa(stat.height) + ", Data: ")
	for {
		tIterValidStart := time.Now()
		if !iter.Valid() {
			break
		}
		tIterValidElasp += time.Since(tIterValidStart)
		tReadStart := time.Now()
		key = iter.GetKey()
		value = iter.GetVal()
		tReadElasp += time.Since(tReadStart)

		dataSize += len(key) + len(value)

		tSha256Start := time.Now()
		//timeFile.WriteString(hex.EncodeToString(key) + hex.EncodeToString(value))
		h.Write(key)
		h.Write(value)
		tSha256Elasp += time.Since(tSha256Start)

		tIterNextStart := time.Now()
		iter.Next()
		tIterNextElasp += time.Since(tIterNextStart)

		i++
	}
	//timeFile.WriteString("\n")
	//for ; iter.Valid(); iter.Next() {
	//	//tReadStart := time.Now()
	//	//key = iter.GetKey()
	//	//value = iter.GetVal()
	//	//tReadElasp += time.Since(tReadStart)
	//	//
	//	//dataSize += len(key) + len(value)
	//	//
	//	//tSha256Start := time.Now()
	//	//h.Write(key)
	//	//h.Write(value)
	//	//tSha256Elasp += time.Since(tSha256Start)
	//	tSha256Start := time.Now()
	//	h.Write(iter.GetKey())
	//	h.Write(iter.GetVal())
	//	tSha256Elasp += time.Since(tSha256Start)
	//
	//	i++
	//}

	tSha256SumStart := time.Now()
	copy(stat.hashSerialized[:], h.Sum(nil))
	tSha256Elasp += time.Since(tSha256SumStart)

	totalStatTime := time.Since(tStatStart)

	timeFile.WriteString(time.Now().String() + ", height: " + strconv.Itoa(stat.height) + ", count: " + strconv.Itoa(i) +
		", size: " + strconv.Itoa(dataSize) + " bytes, read time: " + tReadElasp.String() + ", sha256 time: " +
		tSha256Elasp.String() + ", iter valid time: " + tIterValidElasp.String() + ", iter next time: " +
		tIterNextElasp.String() + ", total time:" + totalStatTime.String() + "\n")
	timeFile.Sync()
	res <- stat.String()
}
