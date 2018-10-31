package wallet

import (
	"github.com/copernet/copernicus/util"
	"io"
	"sync"
)

type AddressBookData struct {
	Account string
	Purpose string
}

type AddressBook struct {
	sync.RWMutex
	addressBook map[string]*AddressBookData
}

func NewEmptyAddressBookData() *AddressBookData {
	return &AddressBookData{}
}

func NewAddressBookData(account string, purpose string) *AddressBookData {
	if purpose == "" {
		purpose = "unknown"
	}
	return &AddressBookData{
		Account: account,
		Purpose: purpose,
	}
}

func (abd *AddressBookData) Serialize(writer io.Writer) error {
	var err error
	if err = util.WriteVarString(writer, abd.Account); err != nil {
		return err
	}
	if err = util.WriteVarString(writer, abd.Purpose); err != nil {
		return err
	}
	return nil
}

func (abd *AddressBookData) SerializeSize() int {
	accountLen := len(abd.Account)
	n := int(util.VarIntSerializeSize(uint64(accountLen)))
	n += accountLen

	purposeLen := len(abd.Purpose)
	n += int(util.VarIntSerializeSize(uint64(purposeLen)))
	n += purposeLen

	return n
}

func (abd *AddressBookData) Unserialize(reader io.Reader) error {
	var err error
	if abd.Account, err = util.ReadVarString(reader); err != nil {
		return err
	}
	if abd.Account, err = util.ReadVarString(reader); err != nil {
		return err
	}
	return nil
}

func (ab *AddressBook) Init() {
	ab.addressBook = make(map[string]*AddressBookData)
}

func (ab *AddressBook) SetAddressBook(keyHash []byte, addressBookData *AddressBookData) {
	ab.Lock()
	defer ab.Unlock()
	ab.addressBook[string(keyHash)] = addressBookData
}

func (ab *AddressBook) GetAccountName(keyHash []byte) string {
	ab.RLock()
	defer ab.RUnlock()
	if addressData, ok := ab.addressBook[string(keyHash)]; ok {
		return addressData.Account
	}
	return ""
}
