package cashaddr

import (
	"testing"

	"github.com/copernet/copernicus/model"
	"reflect"
)

var valid = []string{
	"prefix:x64nx6hz",
	"PREFIX:X64NX6HZ",
	"p:gpf8m4h7",
	"bitcoincash:qpzry9x8gf2tvdw0s3jn54khce6mua7lcw20ayyn",
	"bchtest:testnetaddress4d6njnut",
	"bchreg:555555555555555555555555555555555555555555555udxmlmrz",
}

func TestValid(t *testing.T) {
	for _, s := range valid {
		_, _, err := DecodeCashAddress(s)
		if err != nil {
			t.Error(err)
		}
	}
}

var Invalid = []string{
	"prefix:x32nx6hz",
	"prEfix:x64nx6hz",
	"prefix:x64nx6Hz",
	"pref1x:6m8cxv73",
	"prefix:",
	":u9wsx07j",
	"bchreg:555555555555555555x55555555555555555555555555udxmlmrz",
	"bchreg:555555555555555555555555555555551555555555555udxmlmrz",
	"pre:fix:x32nx6hz",
	"prefixx64nx6hz",
}

func TestInvalid(t *testing.T) {
	for _, s := range Invalid {
		_, _, err := DecodeCashAddress(s)
		if err == nil {
			t.Error("Failed to error on invalid string")
		}
	}
}

func TestDecodeCashAddress(t *testing.T) {
	// Mainnet
	addr, err := DecodeAddress("bitcoincash:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy", &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy" {
		t.Error("Address decoding error")
	}
	addr1, err := DecodeAddress("bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq", &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr1.String() != "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = DecodeAddress("bchtest:qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc", &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = DecodeAddress("bchreg:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7", &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7" {
		t.Error("Address decoding error")
	}
}

var dataElement = []byte{203, 72, 18, 50, 41, 156, 213, 116, 49, 81, 172, 75, 45, 99, 174, 25, 142, 123, 176, 169}

// Second address of https://github.com/Bitcoin-UAHF/spec/blob/master/cashaddr.md#examples-of-address-translation
func TestCashAddressPubKeyHash_EncodeAddress(t *testing.T) {
	// Mainnet
	addr, err := NewCashAddressPubKeyHash(dataElement, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7" {
		t.Error("Address decoding error")
	}
}

var dataElement2 = []byte{118, 160, 64, 83, 189, 160, 168, 139, 218, 81, 119, 184, 106, 21, 195, 178, 159, 85, 152, 115}

// 4th address of https://github.com/Bitcoin-UAHF/spec/blob/master/cashaddr.md#examples-of-address-translation
func TestCashAddressScriptHash_EncodeAddress(t *testing.T) {
	// Mainnet
	addr, err := NewCashAddressScriptHashFromHash(dataElement2, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "ppm2qsznhks23z7629mms6s4cwef74vcwvhanqgjxu" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "ppm2qsznhks23z7629mms6s4cwef74vcwvdp9ptp96" {
		t.Error("Address decoding error")
	}
}

func TestCashAddressPubKeyHash_ScriptAddress(t *testing.T) {
	// MainNet
	addr, err := NewCashAddressPubKeyHash(dataElement, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte := addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement)
	}

	//TestNet
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte = addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement)
	}

	//RegressionNet
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte = addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement)
	}
}

func TestCashAddressScriptHash_ScriptAddress(t *testing.T) {
	// MainNet
	addr, err := NewCashAddressScriptHashFromHash(dataElement2, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte := addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement2) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement2)
	}

	//TestNet
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte = addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement2) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement2)
	}

	//RegressionNet
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	addrByte = addr.ScriptAddress()
	if !reflect.DeepEqual(addrByte, dataElement2) {
		t.Errorf("addrByte:%v not equal dataElement:%v", addrByte, dataElement2)
	}
}
