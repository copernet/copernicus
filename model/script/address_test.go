package script

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/util"
)

func TestPublicKeyToAddress(t *testing.T) {

	publicKey := "03a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bd"
	bytes, err := hex.DecodeString(publicKey)
	if err != nil {
		t.Fatal(err)
		return
	}
	address, err := AddressFromPublicKey(bytes)
	if err != nil {
		t.Fatal(err)
		return
	}
	hash160 := make([]byte, 20)
	copy(hash160, address.hash160[:])
	hash160Hex := hex.EncodeToString(hash160)
	if hash160Hex != "9a1c78a507689f6f54b847ad1cef1e614ee23f1e" {
		t.Errorf("hash160Hex is wrong 9a1c78a507689f6f54b847ad1cef1e614ee23f1e  --  %s", hash160Hex)
		return
	}
	if address.addressStr != "1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV" {
		t.Errorf("address is wrong 1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV  --  %s", address.addressStr)
		return
	}
}

func TestHash160Address(t *testing.T) {
	hash160, err := hex.DecodeString("0000000000000000000000000000000000000000")
	if hex.EncodeToString(hash160) != "0000000000000000000000000000000000000000" {
		t.Error(err)
		return
	}
	address, err := Hash160ToAddressStr(hash160, PublicKeyToAddress)
	if err != nil {
		t.Error(err)
		return
	}
	if address != "1111111111111111111114oLvT2" {
		t.Error("address is worng ,", address)
		return
	}
}

func TestHash160ToAddress(t *testing.T) {
	data, err := hex.DecodeString("0014a4b4ca48de0b3fffc15404a1acdc8dbaae226955")
	if err != nil {
		t.Error(err)
		return
	}
	hash160 := util.Hash160(data)
	if hex.EncodeToString(hash160) != "2928f43af18d2d60e8a843540d8086b305341339" {
		t.Error(err)
		return
	}
	address, err := Hash160ToAddressStr(hash160, ScriptToAddress)
	if err != nil {
		t.Error(err)
		return
	}
	if address != "35SegwitPieWKVHieXd97mnurNi8o6CM73" {
		t.Error("address is worng ,", address)
		return
	}
}

func TestPrivateKeyToAddress(t *testing.T) {
	crypto.InitSecp256()
	address, err := AddressFromPrivateKey("5KYZdUEo39z3FPrtuX2QbbwGnNP5zTd7yyr2SC1j299sBCnWjss")
	if err != nil {
		t.Error(err)
	}
	if address.addressStr != "1HZwkjkeaoZfTSaJxDw6aKkxp45agDiEzN" {
		t.Errorf("address (%s) is error", address.addressStr)
	}

	address, err = AddressFromPrivateKey("L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1")
	if err != nil {
		t.Error(err)
	}
	if address.addressStr != "1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV" {
		t.Errorf("address (%s) is error", address.addressStr)
	}

}

func TestPrivateKeyFromHex(t *testing.T) {

	//str1 := "3714c34e68f8481d"
	//str2 := "9e3647445d5ca65e"
	//str3 := "9d150ddb24d2182a"
	//str4 := "6ac12143f1293835"
	//pri := core.PrivateKeyFromBytes(hexToBytes(str1 + str2 + str3 + str4))
	//if pri == nil {
	//	t.Error("pri is nil")
	//}
	//pub := pri.PubKey()
	//pub.Compressed = true
	//address, err := AddressFromPublicKey(pub.ToBytes())
	//if err != nil {
	//	t.Error(err.Error())
	//}
}

func TestInitAddressParam(t *testing.T) {
	InitAddressParam(&AddressParam{
		PubKeyHashAddressVer: 11,
		ScriptHashAddressVer: 96,
	})
	if AddressVerPubKey() != 11 {
		t.Errorf("TestInitAddressParam test failed, pubKeyAddressVer(%v) not init", AddressVerPubKey())
	}
	if AddressVerScript() != 96 {
		t.Errorf("TestInitAddressParam test failed, scriptAddressVer(%v) not init", AddressVerScript())
	}
	InitAddressParam(&AddressParam{
		PubKeyHashAddressVer: PublicKeyToAddress,
		ScriptHashAddressVer: ScriptToAddress,
	})
	if AddressVerPubKey() != PublicKeyToAddress {
		t.Errorf("TestInitAddressParam test failed, pubKeyAddressVer(%v) not init", AddressVerPubKey())
	}
	if AddressVerScript() != ScriptToAddress {
		t.Errorf("TestInitAddressParam test failed, scriptAddressVer(%v) not init", AddressVerScript())
	}
}

//test address transform: script.AddressFromString
func Test_Address_parse_from_string(t *testing.T) {
	address := "mpLQjfK79b7CCV4VMJWEWAj5Mpx8Up5zxB"

	_, err := AddressFromString(address)

	assert.NoError(t, err)
}

func Test_Address_should_not_be_decoded(t *testing.T) {
	address := ""

	_, err := AddressFromString(address)

	assert.EqualError(t, err, "addressStr length is 0 ,not 25")
}

//test address from scriptHash: script.AddressFromScriptHash
func Test_Address_From_ScriptHash(t *testing.T) {
	scriptBytes, err := createScriptHash()
	if err != nil {
		t.Error("create scripthash error")
	}

	_, err = AddressFromScriptHash(scriptBytes)

	assert.NoError(t, err)
}

func createScriptHash() ([]byte, error) {
	hexScript := "a9146102362b0068a4cd8719e47b4a0d09076a76404387"
	scriptBytes, err := hex.DecodeString(hexScript)
	if err != nil {
		return nil, err
	}
	return scriptBytes, nil
}

// test address form hash160: script.AddressFromHash160
func Test_Address_From_Hash160(t *testing.T) {
	hash160 := createHash160()

	address, err := AddressFromHash160(hash160, crypto.DumpedPrivateKeyVersion)

	assert.NoError(t, err)

	// test some Address function

	version := address.GetVersion()
	assert.Equal(t, int(version), crypto.DumpedPrivateKeyVersion)

	PKHash := address.EncodeToPubKeyHash()
	assert.Equal(t, hash160, PKHash)

	addrStr := address.String()
	t.Log(addrStr)
	address.addressStr = ""
	t.Log(address.String())
}

func createHash160() []byte {
	var keyBytes []byte
	for i := 0; i < 32; i++ {
		keyBytes = append(keyBytes, byte(rand.Uint32()%256))
	}
	privKey := crypto.PrivateKeyFromBytes(keyBytes)
	hash160 := privKey.PubKey().ToHash160()
	return hash160
}
