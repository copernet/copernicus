package script

import (
	"bytes"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/base58"
	"github.com/pkg/errors"
)

const (
	AddressBytesLength       = 25
	Hash160BytesLength       = 20
	PublicKeyToAddressInTest = 0x6f
	PublicKeyToAddress       = 0x00
	ScriptToAddressInTest    = 196
	ScriptToAddress          = 5
)

var IsTestNetwork = false

type Address struct {
	key        *crypto.PrivateKey
	version    byte
	publicKey  []byte
	addressStr string
	hash160    [20]byte
}

func (addr *Address) EncodeToPubKeyHash() []byte {
	return addr.hash160[:]
}

func (addr *Address) String() string {
	if addr.addressStr != "" {
		return addr.addressStr
	}

	return base58.CheckEncode(addr.publicKey, addr.version)
}

func (addr *Address) GetVersion() byte {
	return addr.version
}

func AddressFromString(addressStr string) (btcAddress *Address, err error) {
	decodes := base58.Decode(addressStr) // todo check whether is CheckDecode() or not
	if decodes == nil {
		err = errors.Errorf("can not  base58-decode string  %s", addressStr)
		return
	}
	if len(decodes) != AddressBytesLength {
		err = errors.Errorf("addressStr length is %d ,not %d", len(decodes), AddressBytesLength)
		return
	}
	checkBytes := util.DoubleSha256Bytes(decodes[0:21])
	if !bytes.Equal(checkBytes[:4], decodes[21:25]) {
		err = errors.Errorf("addressStr(%s) checksum failed", addressStr)
	} else {
		var hash160 [20]byte
		copy(hash160[:], decodes[1:21])
		btcAddress = &Address{
			version:    decodes[0],
			hash160:    hash160,
			addressStr: addressStr,
		}
	}
	return
}

func AddressVerPubKey() byte {
	if IsTestNetwork {
		return PublicKeyToAddressInTest
	}
	//if chainparams.ActiveNetParams.Name == "mainnet" {
	//	return PublicKeyToAddress
	//}
	//if chainparams.ActiveNetParams.Name == "regtest" {
	//	return PublicKeyToAddressInTest
	//}
	//if chainparams.ActiveNetParams.Name == "testnet3" {
	//	return PublicKeyToAddressInTest
	//}
	//if chainparams.ActiveNetParams.Name == "simnet" {
	//	return 0x3f
	//}
	return PublicKeyToAddress
}

func AddressVerScript() byte {
	if IsTestNetwork {
		return ScriptToAddressInTest
	}
	return ScriptToAddress
}

func AddressFromHash160(hash160 []byte, version byte) (address *Address, err error) {
	str, err := Hash160ToAddressStr(hash160, version)
	if err != nil {
		return
	}
	var hash160bytes [20]byte
	copy(hash160bytes[:], hash160)
	address = &Address{
		version:    version,
		hash160:    hash160bytes,
		addressStr: str,
	}
	return
}

func Hash160ToAddressStr(hash160 []byte, version byte) (str string, err error) {
	if len(hash160) != Hash160BytesLength {
		err = errors.Errorf("hash160 length %d not %d", len(hash160), Hash160BytesLength)
		return
	}
	result := make([]byte, 25)
	result[0] = version
	copy(result[1:21], hash160)
	checkBytes := util.DoubleSha256Bytes(result[:21])
	copy(result[21:25], checkBytes[:4])
	str = base58.Encode(result)
	return
}

func AddressFromPrivateKey(priKeyStr string) (*Address, error) {
	priKey, err := crypto.DecodePrivateKey(priKeyStr)
	if err != nil {
		return nil, err
	}
	pubKey := priKey.PubKey()
	address, err := AddressFromPublicKey(pubKey.ToBytes())
	if err != nil {
		return nil, err
	}
	address.key = priKey
	return address, nil

}

func AddressFromPublicKey(publicKey []byte) (address *Address, err error) {
	version := AddressVerPubKey()
	address = new(Address)
	address.publicKey = make([]byte, len(publicKey))
	copy(address.publicKey, publicKey)
	address.version = version
	hash160 := util.Hash160(publicKey)
	copy(address.hash160[:], hash160)
	address.addressStr, err = Hash160ToAddressStr(hash160, version)
	return
}

func AddressFromScriptHash(script []byte) (*Address, error) {
	version := AddressVerScript()
	address := new(Address)
	address.publicKey = make([]byte, len(script))
	copy(address.publicKey, script)
	address.version = version
	hash160 := util.Hash160(script)
	copy(address.hash160[:], hash160)
	addressStr, err := Hash160ToAddressStr(hash160, version)
	address.addressStr = addressStr
	return address, err

}
