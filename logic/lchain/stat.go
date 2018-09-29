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
	b := time.Now()
	besthash, err := cdb.GetBestBlock()
	if err != nil {
		log.Debug("in GetUTXOStats, GetBestBlock(), failed=%v\n", err)
		return err
	}
	stat.bestblock = *besthash
	stat.height = int(mchain.GetInstance().FindBlockIndex(*besthash).Height)

	//hashbuf := bytes.NewBuffer(nil)
	//hashbuf.Write(stat.bestblock[:])
	h := sha256.New()
	h.Write(stat.bestblock[:])

	iter := cdb.GetDBW().Iterator(nil)
	defer iter.Close()
	iter.Seek([]byte{db.DbCoin})
	for ; iter.Valid(); iter.Next() {
		//hashbuf.Write(iter.GetKey())
		//hashbuf.Write(iter.GetVal())
		h.Write(iter.GetKey())
		h.Write(iter.GetVal())
	}
	//stat.hashSerialized = util.Sha256Hash(hashbuf.Bytes())
	copy(stat.hashSerialized[:], h.Sum(nil))
	fmt.Printf("iter utxo db time: %v\n", time.Since(b))
	return nil
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
	taskControl = newUtxoTaskControl(1, 20)
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
	f, err := os.OpenFile(filepath.Join(conf.DataDir, "utxo.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0640)
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
	for i := 0; i < tc.numWorker; i++ {
		go func() {
			for {
				select {
				case <-tc.done:
					return
				case arg := <-tc.utxoTask:
					utxoStat(arg.iter, arg.stat, tc.utxoResult)
				}
			}
		}()
	}
}

func utxoStat(iter *db.IterWrapper, stat *stat, res chan<- string) {
	defer iter.Close()
	h := sha256.New()
	h.Write(stat.bestblock[:])
	for ; iter.Valid(); iter.Next() {
		h.Write(iter.GetKey())
		h.Write(iter.GetVal())
	}
	copy(stat.hashSerialized[:], h.Sum(nil))
	res <- stat.String()
}
