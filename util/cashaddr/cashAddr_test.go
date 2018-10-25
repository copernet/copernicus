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
	if addr.String() != "bitcoincash:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy" {
		t.Error("Address decoding error")
	}
	addr1, err := DecodeAddress("bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq", &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr1.String() != "bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = DecodeAddress("bchtest:qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc", &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchtest:qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = DecodeAddress("bchreg:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7", &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchreg:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7" {
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
	if addr.String() != "bitcoincash:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchtest:qr95sy3j9xwd2ap32xkykttr4cvcu7as4ytjg7p7mc" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = NewCashAddressPubKeyHash(dataElement, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchreg:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y3w7lzdc7" {
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
	if addr.String() != "bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq" {
		t.Error("Address decoding error")
	}
	// Testnet
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchtest:ppm2qsznhks23z7629mms6s4cwef74vcwvhanqgjxu" {
		t.Error("Address decoding error")
	}
	// Regtest
	addr, err = NewCashAddressScriptHashFromHash(dataElement2, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchreg:ppm2qsznhks23z7629mms6s4cwef74vcwvdp9ptp96" {
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
	if ok := addr.IsForNet(&model.MainNetParams); !ok {
		t.Errorf("the P2PKH address isn't associated with the bch MainNetWork")
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
	if ok := addr.IsForNet(&model.TestNetParams); !ok {
		t.Errorf("the P2PKH address isn't associated with the bch TestNetWork")
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
	if ok := addr.IsForNet(&model.RegressionNetParams); !ok {
		t.Errorf("the P2PKH address isn't associated with the bch RegressionNetWork")
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
	if ok := addr.IsForNet(&model.MainNetParams); !ok {
		t.Errorf("the P2SH address isn't associated with the bch MainNetWork")
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
	if ok := addr.IsForNet(&model.TestNetParams); !ok {
		t.Errorf("the P2SH address isn't associated with the bch TestNetWork")
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
	if ok := addr.IsForNet(&model.RegressionNetParams); !ok {
		t.Errorf("the P2SH address isn't associated with the bch RegressionNetWork")
	}
}

func TestNewCashAddressScriptHash(t *testing.T) {
	//MainNet
	addr, err := NewCashAddressScriptHash(dataElement2, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bitcoincash:pqgml2l4azsajnfqp30pmetax3ek9wc9lvghvvkx6d" {
		t.Error("Address decoding error")
	}

	//TestNet
	addr, err = NewCashAddressScriptHash(dataElement2, &model.TestNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchtest:pqgml2l4azsajnfqp30pmetax3ek9wc9lvv9gt53a3" {
		t.Error("Address decoding error")
	}

	//RegressionNet
	addr, err = NewCashAddressScriptHash(dataElement2, &model.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	if addr.String() != "bchreg:pqgml2l4azsajnfqp30pmetax3ek9wc9lvke72hz7h" {
		t.Error("Address decoding error")
	}
}

var (
	P2SHPKScript  = []byte{169, 20, 118, 160, 64, 83, 189, 160, 168, 139, 218, 81, 119, 184, 106, 21, 195, 178, 159, 85, 152, 115, 135}
	P2PKHPKScript = []byte{118, 169, 20, 203, 72, 18, 50, 41, 156, 213, 116, 49, 81, 172, 75, 45, 99, 174, 25, 142, 123, 176, 169, 136, 172}
	ErrorPKScript = []byte{20, 118, 160, 64, 83, 189, 160, 168, 139, 218, 81, 119, 184, 106, 21, 195, 178, 159, 85, 152, 115}
)

func TestExtractPkScriptAddrs(t *testing.T) {
	p2shAddr, err := ExtractPkScriptAddrs(P2SHPKScript, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if p2shAddr.String() != "bitcoincash:ppm2qsznhks23z7629mms6s4cwef74vcwvn0h829pq" {
		t.Errorf("p2shAddr:%s parse error", p2shAddr.String())
	}

	p2pkhAddr, err := ExtractPkScriptAddrs(P2PKHPKScript, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	if p2pkhAddr.String() != "bitcoincash:qr95sy3j9xwd2ap32xkykttr4cvcu7as4y0qverfuy" {
		t.Errorf("p2pkhAddr:%s parse error", p2pkhAddr.String())
	}

	errAddr, err := ExtractPkScriptAddrs(ErrorPKScript, &model.MainNetParams)
	if err == nil {
		t.Error(err)
	}
	if errAddr != nil {
		t.Errorf("the errAddr:%s should equal nil.", errAddr.String())
	}
}

func TestCashPayToAddrScript(t *testing.T) {
	addr, err := NewCashAddressPubKeyHash(dataElement, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	pkScript, err := CashPayToAddrScript(addr)
	if err != nil {
		t.Error(err)
	}
	if !(len(pkScript) == 1+1+1+20+1+1 && pkScript[0] == 0x76 && pkScript[1] == 0xa9 && pkScript[2] == 0x14 && pkScript[23] == 0x88 && pkScript[24] == 0xac) {
		t.Errorf("the pkScript: %v is not p2pkh script", pkScript)
	}

	addr1, err := NewCashAddressScriptHashFromHash(dataElement2, &model.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	p2pkhScript, err := CashPayToAddrScript(addr1)
	if err != nil {
		t.Error(err)
	}

	if !(len(p2pkhScript) == 1+1+20+1 && p2pkhScript[0] == 0xa9 && p2pkhScript[1] == 0x14 && p2pkhScript[22] == 0x87) {
		t.Errorf("the pkScript: %v is not p2sh script", p2pkhScript)
	}
}
