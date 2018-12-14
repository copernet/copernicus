package wallet

import (
	"bytes"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

type WalletDB struct {
	*db.DBWrapper
}

func (wdb *WalletDB) initDB() {
	walletDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/wallet",
		CacheSize: (1 << 20) * 8,
		Wipe:      false,
	}

	var err error
	if wdb.DBWrapper, err = db.NewDBWrapper(walletDbCfg); err != nil {
		panic("init wallet DB failed..." + err.Error())
	}
}

func (wdb *WalletDB) loadSecrets() [][]byte {
	itr := wdb.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletKey})

	secrets := make([][]byte, 0)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletKey; itr.Next() {
		secrets = append(secrets, itr.GetKey()[1:])
	}
	return secrets
}

func (wdb *WalletDB) loadScripts() ([]*script.Script, error) {
	itr := wdb.Iterator(nil)
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
	itr := wdb.Iterator(nil)
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
	itr := wdb.Iterator(nil)
	defer itr.Close()
	itr.Seek([]byte{db.DbWalletTx})

	txns := make([]*WalletTx, 0)
	for ; itr.Valid() && itr.GetKey()[0] == db.DbWalletTx; itr.Next() {
		walletTx := NewEmptyWalletTx()
		if err := walletTx.Unserialize(bytes.NewBuffer(itr.GetVal())); err != nil {
			return nil, err
		}
		walletTx.spentStatus = make([]bool, walletTx.GetOutsCount())
		txns = append(txns, walletTx)
	}
	return txns, nil
}

func (wdb *WalletDB) saveSecret(secret []byte) error {
	key := getDBKey(db.DbWalletKey, secret)
	return wdb.Write(key, []byte{}, true)
}

func (wdb *WalletDB) saveScript(sc *script.Script) error {
	w := new(bytes.Buffer)
	err := sc.Serialize(w)
	if err != nil {
		return err
	}

	key := getDBKey(db.DbWalletScript, w.Bytes())
	return wdb.Write(key, []byte{}, true)
}

func (wdb *WalletDB) saveAddressBook(keyHash []byte, data *AddressBookData) error {
	w := new(bytes.Buffer)
	err := data.Serialize(w)
	if err != nil {
		return err
	}

	key := getDBKey(db.DbWalletAddrBook, keyHash)
	return wdb.Write(key, w.Bytes(), true)
}

func (wdb *WalletDB) saveWalletTx(wtx *WalletTx) error {
	w := new(bytes.Buffer)
	err := wtx.Serialize(w)
	if err != nil {
		return err
	}

	txHash := wtx.GetHash()
	key := getDBKey(db.DbWalletTx, txHash[:])
	return wdb.Write(key, w.Bytes(), true)
}

func (wdb *WalletDB) removeWalletTx(txHash *util.Hash) error {
	key := getDBKey(db.DbWalletTx, txHash[:])
	return wdb.Erase(key, true)
}

func getDBKey(dbID byte, orgKey []byte) []byte {
	dbKey := make([]byte, 0, len(orgKey)+1)
	dbKey = append(dbKey, dbID)
	dbKey = append(dbKey, orgKey...)
	return dbKey
}
