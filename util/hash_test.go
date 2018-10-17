package util

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var siphash24TestVec = [...]uint64{
	0x726fdb47dd0e0e31, 0x74f839c593dc67fd, 0x0d6c8009d9a94f5a,
	0x85676696d7fb7e2d, 0xcf2794e0277187b7, 0x18765564cd99a68d,
	0xcbc9466e58fee3ce, 0xab0200f58b01d137, 0x93f5f5799a932462,
	0x9e0082df0ba9e4b0, 0x7a5dbbc594ddb9f3, 0xf4b32f46226bada7,
	0x751e8fbc860ee5fb, 0x14ea5627c0843d90, 0xf723ca908e7af2ee,
	0xa129ca6149be45e5, 0x3f2acc7f57c29bdb, 0x699ae9f52cbe4794,
	0x4bc1b3f0968dd39c, 0xbb6dc91da77961bd, 0xbed65cf21aa2ee98,
	0xd0f2cbb02e3b67c7, 0x93536795e3a33e88, 0xa80c038ccd5ccec8,
	0xb8ad50c6f649af94, 0xbce192de8a85b8ea, 0x17d835b85bbb15f3,
	0x2f2e6163076bcfad, 0xde4daaaca71dc9a5, 0xa6a2506687956571,
	0xad87a3535c49ef28, 0x32d892fad841c342, 0x7127512f72f27cce,
	0xa7f32346f95978e3, 0x12e0b01abb051238, 0x15e034d40fa197ae,
	0x314dffbe0815a3b4, 0x027990f029623981, 0xcadcd4e59ef40c4d,
	0x9abfd8766a33735c, 0x0e3ea96b5304a7d0, 0xad0c42d6fc585992,
	0x187306c89bc215a9, 0xd4a60abcf3792b95, 0xf935451de4f21df2,
	0xa9538f0419755787, 0xdb9acddff56ca510, 0xd06c98cd5c0975eb,
	0xe612a3cb9ecba951, 0xc766e62cfcadaf96, 0xee64435a9752fe72,
	0xa192d576b245165a, 0x0a8787bf8ecb74b2, 0x81b3e73d20b49b6f,
	0x7fa8220ba3b2ecea, 0x245731c13ca42499, 0xb78dbfaf3a8d83bd,
	0xea1ad565322a1a0b, 0x60e61c23a3795013, 0x6606d7e446282b93,
	0x6ca4ecb15c5f91e1, 0x9f626da15c9625f3, 0xe51b38608ef25f57,
	0x958a324ceb064572,
}

func TestSipHash(t *testing.T) {
	sip := NewSipHasher(0x0706050403020100, 0x0F0E0D0C0B0A0908)
	if result := sip.Finalize(); result != 0x726fdb47dd0e0e31 {
		t.Errorf("expect Finalize() = 0x726fdb47dd0e0e31, but got result = 0x%08x", result)
	}
	if result := sip.Write([]byte{0}).Finalize(); result != 0x74f839c593dc67fd {
		t.Errorf("expect Finalize() = 0x74f839c593dc67fd, but go result = 0x%08x", result)
	}
	if result := sip.Write([]byte{1, 2, 3, 4, 5, 6, 7}).Finalize(); result != 0x93f5f5799a932462 {
		t.Errorf("expect Finalize() = 0x93f5f5799a932462, but go result = 0x%08x", result)
	}
	if result := sip.WriteUint64(0x0F0E0D0C0B0A0908).Finalize(); result != 0x3f2acc7f57c29bdb {
		t.Errorf("expect Finalize() = 0x3f2acc7f57c29bdb, but go result = 0x%08x", result)
	}
	if result := sip.Write([]byte{16, 17}).Finalize(); result != 0x4bc1b3f0968dd39c {
		t.Errorf("expect Finalize() = 0x4bc1b3f0968dd39c, but go result = 0x%08x", result)
	}
	if result := sip.Write([]byte{18, 19, 20, 21, 22, 23, 24, 25, 26}).Finalize(); result != 0x2f2e6163076bcfad {
		t.Errorf("expect Finalize() = 0x2f2e6163076bcfad, but go result = 0x%08x", result)
	}
	if result := sip.Write([]byte{27, 28, 29, 30, 31}).Finalize(); result != 0x7127512f72f27cce {
		t.Errorf("expect Finalize() = 0x7127512f72f27cce, but go result = 0x%08x", result)
	}
	if result := sip.WriteUint64(0x2726252423222120).Finalize(); result != 0x0e3ea96b5304a7d0 {
		t.Errorf("expect Finalize() = 0x0e3ea96b5304a7d0, but go result = 0x%08x", result)
	}
	if result := sip.WriteUint64(0x2F2E2D2C2B2A2928).Finalize(); result != 0xe612a3cb9ecba951 {
		t.Errorf("expect Finalize() = 0xe612a3cb9ecba951, but go result = 0x%08x", result)
	}

	h, err := GetHashFromStr("1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100")
	if err != nil {
		t.Errorf("invalid hash string")
	}
	if result := SipHash(0x0706050403020100, 0x0F0E0D0C0B0A0908, (*h)[:]); result != 0x7127512f72f27cce {
		t.Errorf("expect SipHash() = 0x7127512f72f27cce, but go result = 0x%08x", result)
	}

	sip2 := NewSipHasher(0x0706050403020100, 0x0F0E0D0C0B0A0908)
	for i := byte(0); i < byte(len(siphash24TestVec)); i++ {
		if result := sip2.Finalize(); result != siphash24TestVec[i] {
			t.Errorf("expect Finalize() = 0x%08x, but go result = 0x%08x", siphash24TestVec[i], result)
		}
		sip2.Write([]byte{i})
	}

	sip3 := NewSipHasher(0x0706050403020100, 0x0F0E0D0C0B0A0908)
	for i := 0; i < len(siphash24TestVec); i += 8 {
		if result := sip3.Finalize(); result != siphash24TestVec[i] {
			t.Errorf("expect Finalize() = 0x%08x, but go result = 0x%08x", siphash24TestVec[i], result)
		}
		sip3.WriteUint64(uint64(i) | (uint64(i+1) << 8) |
			(uint64(i+2) << 16) | (uint64(i+3) << 24) |
			(uint64(i+4) << 32) | (uint64(i+5) << 40) |
			(uint64(i+6) << 48) | (uint64(i+7) << 56))
	}
}

func TestHash_IsEqual(t *testing.T) {
	hash1 := HashFromString("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")
	hash2 := HashFromString("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")
	if !hash1.IsEqual(hash2) {
		t.Errorf("IsEqual test failed")
		return
	}
}

func TestHashObjectToString(t *testing.T) {
	hash := *HashFromString("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")
	s1 := fmt.Sprintf("hash: %s", hash)
	s2 := fmt.Sprintf("hash: %v", hash)
	s3 := fmt.Sprintf("hash: %s", hash.String())

	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s1)
	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s2)
	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s3)
}

func TestHashPointerToString(t *testing.T) {
	hash := HashFromString("00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721")
	s1 := fmt.Sprintf("hash: %s", hash)
	s2 := fmt.Sprintf("hash: %v", hash)
	s3 := fmt.Sprintf("hash: %s", hash.String())

	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s1)
	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s2)
	assert.Equal(t, "hash: 00000000000743f190a18c5577a3c2d2a1f610ae9601ac046a38084ccb7cd721", s3)
}

func TestHash_IsNull(t *testing.T) {
	tests := []struct {
		hash Hash
		want bool
	}{
		{HashZero, true},
		{HashOne, false},
	}

	for i, v := range tests {
		value := v
		result := value.hash.IsNull()
		if result != value.want {
			t.Errorf("The %d value is not expect.", i)
		}
	}
}

func TestHash_Cmp(t *testing.T) {
	tests := []struct {
		hash     *Hash
		target   *Hash
		wantBool bool
		wantNum  int
	}{
		{&HashZero, &HashZero, true, 0},
		{&HashZero, &HashOne, false, -1},
		{nil, nil, true, 0},
		{&HashZero, nil, false, 0},
	}

	for i, v := range tests {
		value := v
		result := value.hash.Cmp(value.target)
		if result != value.wantNum {
			t.Errorf("The %d value exec function Cmp is not expect.", i)
		}

		flag := value.hash.IsEqual(value.target)
		if flag != value.wantBool {
			t.Errorf("The %d value exec function IsEqual is not expect.", i)
		}

	}
}

func TestHash_EncodeDecode(t *testing.T) {
	tests := []struct {
		hash Hash
		want int
	}{
		{HashZero, int(HashZero.SerializeSize())},
	}

	for i, v := range tests {
		value := v
		var buf bytes.Buffer
		result, err := value.hash.Serialize(&buf)
		if err != nil {
			t.Error(err)
		}
		if result != value.want {
			t.Errorf("The %d value is not expect.", i)
		}

		readResult, err := value.hash.Unserialize(&buf)
		if err != nil {
			t.Error(err)
		}
		if readResult != value.want {
			t.Errorf("The %d value is not expect.", i)
		}
	}
}

func TestFunctions(t *testing.T) {
	HashZero.SerializeSize()

	var a []byte
	Sha256Bytes(a)
	Sha256Hash(a)
	DoubleSha256Bytes(a)
	DoubleSha256Hash(a)
	Hash160(a)
	Ripemd160(a)
	Sha1(a)
}
