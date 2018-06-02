package utxo

import (
	"fmt"
	"testing"
	
	"github.com/btcboost/copernicus/persist/db"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestMain(m *testing.M){
	config := UtxoConfig{Do: &db.DBOption{CacheSize: 10000}}
	InitUtxoLruTip(&config)
	m.Run()
}
// func TestNewCoinsLruCache(t *testing.T) {
//
// 	coinsmap := NewEmptyCoinsMap()
// 	coin := NewEmptyCoin()
// 	coin.isCoinBase = true
// 	coin.height = 100
// 	h := util.HashZero
// 	op := outpoint.NewOutPoint(h, 0)
// 	coinsmap.AddCoin(op, coin)
// 	utxoLruTip.UpdateCoins(coinsmap, &util.HashZero)
// 	utxoLruTip.Flush()
// 	c := utxoLruTip.GetCoin(op)
// 	fmt.Println("c==TestNewCoinsLruCache=====%#v", c)
// }
//
// func TestCoinsLruCache_GetCoin(t *testing.T) {
// 	h := util.HashZero
// 	op := outpoint.NewOutPoint(h, 0)
// 	c := utxoLruTip.GetCoin(op)
// 	fmt.Println("c===TestCoinsLruCache_GetCoin====%#v", c)
// }


func TestCoinsDB_BatchWrite(t *testing.T) {
	// d := utxoLruTip.(*CoinsLruCache).db
	// batch := db.NewBatchWrapper(d.dbw)
	// tk := []byte("aaaaaaaaaa")
	// tv := []byte("bbbbbbbbbb")
	// batch.Write(tk, tv)
	// err := d.dbw.WriteBatch(batch, true)
	// fmt.Println(err)
	// v, e := d.dbw.Read(tv)
	// fmt.Println(v,e)
	// tk1 := []byte("cccccccccc")
	// tv1 := []byte("dddddddddd")
	// e = d.dbw.Write(tk1,tv1, true)
	// fmt.Println(e)
	// v1, e1 := d.dbw.Read(tv1)
	// fmt.Println(v1,e1)
	
	//
	ldb,_ := leveldb.OpenFile("/tmp/db", nil)
	// tk2 := []byte("ffffffffff")
	// tv2 := []byte("ffffffffff")
	// data, e2:= ldb.Get(tk2,nil)
	// fmt.Println(data,e2)
	// ldb.Put(tk2, tv2, nil)
	// data, e2 = ldb.Get(tk2,nil)
	// fmt.Println(data,e2)
	
	
	
	bat := new(leveldb.Batch)
	tk3 := []byte("g")
	tv3 := []byte("g")
	tk4 := []byte("h")
	d3,e3 := ldb.Get(tk3, nil)
	d4,e4 := ldb.Get(tk4, nil)
	fmt.Println(d3,d4,e3,e4)
	bat.Put(tk3,tv3)
	bat.Put(tk4,tv3)
	ldb.Write(bat, nil)
	d3,e3 = ldb.Get(tk3, nil)
	d4,e4 = ldb.Get(tk4, nil)
	fmt.Println(d3,d4,e3,e4)
}