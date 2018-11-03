package wallet

import (
	"bytes"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

type WalletDB struct {
	dbw *db.DBWrapper
}

func (wdb *WalletDB) initDB() {
	walletDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/wallet",
		CacheSize: (1 << 20) * 8,
		Wipe:      false,
	}

	var err error
	if wdb.dbw, err = db.NewDBWrapper(walletDbCfg); err != nil {
		panic("init wallet DB failed..." + err.Error())
	}
}

func (wdb *WalletDB) loadSecrets() [][]byte {
	itr := wdb.dbw.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletKey})

	secrets := make([][]byte, 0)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletKey; itr.Next() {
		secrets = append(secrets, itr.GetKey()[1:])
	}
	return secrets
}

func (wdb *WalletDB) loadScripts() ([]*script.Script, error) {
	itr := wdb.dbw.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletScript})

	scripts := make([]*script.Script, 0)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletScript; itr.Next() {
		sc := script.NewEmptyScript()
		if err := sc.Unserialize(bytes.NewBuffer(itr.GetKey()[1:]), false); err != nil {
			return nil, err
		}
		scripts = append(scripts, sc)
	}
	return scripts, nil
}

func (wdb *WalletDB) loadAddressBook() (map[string]*AddressBookData, error) {
	itr := wdb.dbw.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletAddrBook})

	addressBook := make(map[string]*AddressBookData)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletAddrBook; itr.Next() {
		data := NewEmptyAddressBookData()
		if err := data.Unserialize(bytes.NewBuffer(itr.GetVal())); err != nil {
			return nil, err
		}
		addressBook[string(itr.GetKey()[1:])] = data
	}
	return addressBook, nil
}

func (wdb *WalletDB) loadTransactions() ([]*WalletTx, error) {
	itr := wdb.dbw.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletTx})

	txns := make([]*WalletTx, 0)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletTx; itr.Next() {
		walletTx := NewEmptyWalletTx()
		if err := walletTx.Unserialize(bytes.NewBuffer(itr.GetVal())); err != nil {
			return nil, err
		}
		txns = append(txns, walletTx)
	}
	return txns, nil
}

func (wdb *WalletDB) saveSecret(secrets []byte) error {
	key := make([]byte, 0, len(secrets)+1)
	key = append(key, db.DbWalletKey)
	key = append(key, secrets...)
	return wdb.dbw.Write(key, []byte{}, true)
}

func (wdb *WalletDB) saveScript(sc *script.Script) error {
	w := new(bytes.Buffer)
	err := sc.Serialize(w)
	if err != nil {
		return err
	}

	key := make([]byte, 0, sc.SerializeSize()+1)
	key = append(key, db.DbWalletScript)
	key = append(key, w.Bytes()...)
	return wdb.dbw.Write(key, []byte{}, true)
}

func (wdb *WalletDB) saveAddressBook(keyHash []byte, data *AddressBookData) error {
	w := new(bytes.Buffer)
	err := data.Serialize(w)
	if err != nil {
		return err
	}

	key := make([]byte, 0, len(keyHash)+1)
	key = append(key, db.DbWalletAddrBook)
	key = append(key, keyHash...)
	return wdb.dbw.Write(key, w.Bytes(), true)
}

func (wdb *WalletDB) saveWalletTx(txHash *util.Hash, wtx *WalletTx) error {
	w := new(bytes.Buffer)
	err := wtx.Serialize(w)
	if err != nil {
		return err
	}

	hashBytes := txHash[:]
	key := make([]byte, 0, len(hashBytes)+1)
	key = append(key, db.DbWalletTx)
	key = append(key, hashBytes...)
	return wdb.dbw.Write(key, w.Bytes(), true)
}

func (wdb *WalletDB) removeWalletTx(txHash *util.Hash) error {

	hashBytes := txHash[:]
	key := make([]byte, 0, len(hashBytes)+1)
	key = append(key, db.DbWalletTx)
	key = append(key, hashBytes...)
	return wdb.dbw.Erase(key, true)
}
