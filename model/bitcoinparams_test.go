package model

import (
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/model/script"
	"github.com/stretchr/testify/assert"
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
	ActiveNetParams = &MainNetParams
	assert.Equal(t, MainNetParams.PubKeyHashAddressID, script.AddressVerPubKey())
	assert.Equal(t, MainNetParams.ScriptHashAddressID, script.AddressVerScript())
	assert.True(t, IsPublicKeyHashAddressID(MainNetParams.PubKeyHashAddressID))
	assert.True(t, IsScriptHashAddressid(MainNetParams.ScriptHashAddressID))

	SetTestNetParams()
	assert.Equal(t, TestNetParams.PubKeyHashAddressID, script.AddressVerPubKey())
	assert.Equal(t, TestNetParams.ScriptHashAddressID, script.AddressVerScript())
	assert.True(t, IsPublicKeyHashAddressID(TestNetParams.PubKeyHashAddressID))
	assert.True(t, IsScriptHashAddressid(TestNetParams.ScriptHashAddressID))

	SetRegTestParams()
	assert.Equal(t, RegressionNetParams.PubKeyHashAddressID, script.AddressVerPubKey())
	assert.Equal(t, RegressionNetParams.ScriptHashAddressID, script.AddressVerScript())
	assert.True(t, IsPublicKeyHashAddressID(RegressionNetParams.PubKeyHashAddressID))
	assert.True(t, IsScriptHashAddressid(RegressionNetParams.ScriptHashAddressID))
}

func TestHDPrivateKeyToPublicKeyID(t *testing.T) {
	tests := []struct {
		name       string
		hdPriKeyID []byte
		hdPubKeyID []byte
		isOK       bool
	}{
		{
			name:       "mainnet",
			hdPriKeyID: MainNetParams.HDPrivateKeyID[:],
			hdPubKeyID: MainNetParams.HDPublicKeyID[:],
			isOK:       true,
		},
		{
			name:       "testnet",
			hdPriKeyID: TestNetParams.HDPrivateKeyID[:],
			hdPubKeyID: TestNetParams.HDPublicKeyID[:],
			isOK:       true,
		},
		{
			name:       "regtest",
			hdPriKeyID: RegressionNetParams.HDPrivateKeyID[:],
			hdPubKeyID: RegressionNetParams.HDPublicKeyID[:],
			isOK:       true,
		},
		{
			name:       "unknown",
			hdPriKeyID: []byte{0},
			hdPubKeyID: []byte{0},
			isOK:       false,
		},
	}
	for _, test := range tests {
		t.Logf("testing net:%s", test.name)
		hdPubKeyID, err := HDPrivateKeyToPublicKeyID(test.hdPriKeyID)
		if test.isOK {
			assert.Nil(t, err)
			assert.Equal(t, test.hdPubKeyID, hdPubKeyID)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestIsUAHFEnabled(t *testing.T) {
	ActiveNetParams = &MainNetParams

	isEnable := IsUAHFEnabled(0)
	assert.False(t, isEnable)

	isEnable = IsUAHFEnabled(MainNetParams.UAHFHeight)
	assert.True(t, isEnable)
}

func TestIsMonolithEnabled(t *testing.T) {
	ActiveNetParams = &MainNetParams

	isEnable := IsMonolithEnabled(0)
	assert.False(t, isEnable)

	isEnable = IsMonolithEnabled(MainNetParams.MonolithActivationTime)
	assert.True(t, isEnable)
}

func TestIsDAAEnabled(t *testing.T) {
	ActiveNetParams = &MainNetParams

	isEnable := IsDAAEnabled(0)
	assert.False(t, isEnable)

	isEnable = IsDAAEnabled(MainNetParams.DAAHeight)
	assert.True(t, isEnable)
}

func TestIsReplayProtectionEnabled(t *testing.T) {
	ActiveNetParams = &MainNetParams

	isEnable := IsReplayProtectionEnabled(0)
	assert.False(t, isEnable)

	isEnable = IsReplayProtectionEnabled(MainNetParams.MagneticAnomalyActivationTime)
	assert.True(t, isEnable)
}

func TestGetBlockSubsidy(t *testing.T) {
	netParams := &MainNetParams
	halfSubsidyHeight := netParams.SubsidyReductionInterval
	nonSubsidyHeight := netParams.SubsidyReductionInterval * 64

	tests := []struct {
		name   string
		height int32
		expect float64
	}{
		{
			name:   "genesis",
			height: 0,
			expect: 50,
		},
		{
			name:   "first half subsidy",
			height: halfSubsidyHeight,
			expect: 25,
		},
		{
			name:   "no subsidy",
			height: nonSubsidyHeight,
			expect: 0,
		},
	}
	for _, test := range tests {
		t.Logf("testing case:%s", test.name)
		amt := GetBlockSubsidy(test.height, netParams)
		assert.Equal(t, test.expect, amt.ToBTC())

	}
}
