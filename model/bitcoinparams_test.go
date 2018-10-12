package model

import (
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/model/script"
	"testing"
	"time"
)

func TestBitcoinParamsTxData(t *testing.T) {
	hash := TestNetGenesisBlock.Header.GetHash()
	fmt.Println("genesis hash : ", hash.String())
	fmt.Println("hash 000 : ", hash.String())
	fmt.Println("hash : ", hex.EncodeToString(TestNetGenesisHash[:]))

	fmt.Println("time : ", time.Unix(1296688602, 0).UnixNano())
}

func TestInitActiveNetAddressParams(t *testing.T) {
	setActiveNetAddressParams()
	if script.AddressVerPubKey() != MainNetParams.PubKeyHashAddressID {
		t.Errorf("TestInitAddressParam test failed, mainnet pubKeyAddressVer(%v) not init", script.AddressVerPubKey())
	}
	if script.AddressVerScript() != MainNetParams.ScriptHashAddressID {
		t.Errorf("TestInitAddressParam test failed, mainnet scriptAddressVer(%v) not init", script.AddressVerScript())
	}

	SetTestNetParams()
	if script.AddressVerPubKey() != TestNetParams.PubKeyHashAddressID {
		t.Errorf("TestInitAddressParam test failed, testnet pubKeyAddressVer(%v) not init", script.AddressVerPubKey())
	}
	if script.AddressVerScript() != TestNetParams.ScriptHashAddressID {
		t.Errorf("TestInitAddressParam test failed, testnet scriptAddressVer(%v) not init", script.AddressVerScript())
	}

	SetRegTestParams()
	if script.AddressVerPubKey() != RegressionNetParams.PubKeyHashAddressID {
		t.Errorf("TestInitAddressParam test failed, regtest pubKeyAddressVer(%v) not init", script.AddressVerPubKey())
	}
	if script.AddressVerScript() != RegressionNetParams.ScriptHashAddressID {
		t.Errorf("TestInitAddressParam test failed, regtest scriptAddressVer(%v) not init", script.AddressVerScript())
	}
}
