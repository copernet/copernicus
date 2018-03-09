package crypto

import (
	"bytes"
	"testing"
)

//the sigScript from TxID : fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4
var validSig = []byte{
	0x30,
	0x46, //sigLenth
	0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
	0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
	0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
	0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
	0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
	0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
	0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
	0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
	0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
	0x01, //sigType
}

func TestIsValidSignatureEncoding(t *testing.T) {
	ret := IsValidSignatureEncoding(validSig)
	if !ret {
		t.Error("the test signature is valid")
	}

	ret = IsDefineHashtypeSignature(validSig)
	if !ret {
		t.Error("the test signature hashType is valid")
	}

}

func TestCheckSignatureEncoding(t *testing.T) {
	flags := SCRIPT_VERIFY_STRICTENC
	ret, err := CheckSignatureEncoding(validSig, uint32(flags))
	if err != nil || !ret {
		t.Error("the test signature is valid, ", err)
	}

	flags = SCRIPT_VERIFY_DERSIG
	ret, err = CheckSignatureEncoding(validSig, uint32(flags))
	if err != nil || !ret {
		t.Error("the test signature is valid, ", err)
	}
}

func TestParseSignature(t *testing.T) {
	sig := validSig[:len(validSig)-1]
	signature, err := ParseDERSignature(sig)
	if err != nil {
		t.Error(err)
	}

	sigByte := signature.Serialize()
	if !bytes.Equal(sigByte, sig) {
		t.Errorf("the new serialize signature %v should be equal origin sig %v: ", sigByte, sig)
	}
}
