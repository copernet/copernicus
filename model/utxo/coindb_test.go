package utxo

import (
	"bytes"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"reflect"
	"testing"
)

func TestCoinsDB(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})
	path, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		t.Fatal(err)
	}

	dbo := db.DBOption{
		FilePath:       path,
		CacheSize:      1 << 20,
		Wipe:           false,
		DontObfuscate:  false,
		ForceCompactdb: false,
	}

	dbObj := newCoinsDB(&dbo)

	dbw := dbObj.GetDBW()
	if !reflect.DeepEqual(dbw, dbObj.dbw) {
		t.Errorf("dbw get by GetDBW is not equal to obj.dbw")
	}

	chain.InitGlobalChain()

	if err != nil {
		t.Fatal(err)
	}

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	if dbObj.HaveCoin(&outpoint1) {
		t.Errorf("the db not have coin")
	}

	if _, err := dbObj.GetCoin(&outpoint1); err != leveldb.ErrNotFound {
		t.Errorf("the db not have coin, so the coin is nil.")
	}

	bestBlockHash, err := dbObj.GetBestBlock()
	if err != leveldb.ErrNotFound {
		t.Errorf("there should be none bestblock")
	}

	hashBytes := bytes.NewBuffer(nil)
	hash1.Encode(hashBytes)
	batch := db.NewBatchWrapper(dbObj.dbw)
	batch.Write([]byte{db.DbBestBlock}, hashBytes.Bytes())
	err = dbObj.dbw.WriteBatch(batch, true)
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err = dbObj.GetBestBlock()
	if err != nil {
		t.Fatal(err)
	}

	if bestBlockHash.String() == "000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6" {
		t.Logf("get right best block hash")
	} else {
		t.Fatalf("best block hash should equal to 000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	}

	size := dbObj.EstimateSize()
	t.Logf("db size: %d", size)

	err = os.RemoveAll(path)
	if err != nil {
		t.Fatalf("clean temp directory failed: %s", path)
	}

}
