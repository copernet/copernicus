package util

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math/big"

	"github.com/copernet/copernicus/log"
	"golang.org/x/crypto/ripemd160"
)

const (
	Hash256Size       = 32
	MaxHashStringSize = Hash256Size * 2
	Hash160Size       = 20
)

type Hash [Hash256Size]byte

var HashZero = Hash{0x0000000000000000000000000000000000000000000000000000000000000000}
var HashOne = Hash{0x0000000000000000000000000000000000000000000000000000000000000001}

func Sha256Bytes(b []byte) []byte {
	hash := sha256.Sum256(b)
	return hash[:Hash256Size]
}
func Sha256Hash(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

func DoubleSha256Bytes(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}
func DoubleSha256Hash(b []byte) Hash {
	first := sha256.Sum256(b)
	return Hash(sha256.Sum256(first[:Hash256Size]))
}

// Hash160 calculates the hash ripemd160(sha256(b)).
func Hash160(buf []byte) []byte {
	return calcHash(calcHash(buf, sha256.New()), ripemd160.New())
}

func Ripemd160(buf []byte) []byte {
	return calcHash(buf, ripemd160.New())
}

// Calculate the hash of hasher over buf.
func calcHash(buf []byte, hasher hash.Hash) []byte {
	hasher.Write(buf)
	return hasher.Sum(nil)
}

func Sha1(buf []byte) [20]byte {
	return sha1.Sum(buf)
}

func (hash *Hash) String() string {
	bytes := hash.GetCloneBytes()
	for i := 0; i < Hash256Size/2; i++ {
		bytes[i], bytes[Hash256Size-1-i] = bytes[Hash256Size-1-i], bytes[i]
	}
	return hex.EncodeToString(bytes[:])
}

func (hash *Hash) SerializeSize() uint32 {
	return hash.EncodeSize()
}

func (hash *Hash) Serialize(w io.Writer) (int, error) {
	return hash.Encode(w)
}

func (hash *Hash) Unserialize(r io.Reader) (int, error) {
	return hash.Decode(r)
}

func (hash *Hash) EncodeSize() uint32 {
	return Hash256Size
}

func (hash *Hash) Encode(w io.Writer) (int, error) {
	length, err := w.Write(hash[:])
	if length != Hash256Size || err != nil {
		log.Alert("hash.Unserialize err: ", length, err)
		return length, err
	}
	return length, err
}

func (hash *Hash) Decode(r io.Reader) (int, error) {
	length, err := io.ReadFull(r, hash[:])
	if length != Hash256Size || err != nil {
		log.Alert("hash.Unserialize err: ", length, err)
		return length, err
	}
	return length, err
}

func (hash *Hash) GetCloneBytes() []byte {
	bytes := make([]byte, Hash256Size)
	copy(bytes, hash[:])
	return bytes
}
func (hash *Hash) ToBigInt() *big.Int {
	return new(big.Int).SetBytes(hash.GetCloneBytes())
}

func (hash *Hash) Cmp(other *Hash) int {

	if hash == nil || other == nil {
		return 0
	} else if hash == nil {
		return -1
	} else if other == nil {
		return 1
	}
	return hash.ToBigInt().Cmp(other.ToBigInt())
}

func (hash *Hash) IsEqual(target *Hash) bool {
	if hash == nil && target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}

func (hash *Hash) IsNull() bool {
	for _, item := range hash {
		if item != 0 {
			return false
		}
	}
	return true
}

func HashFromString(hexString string) *Hash {
	hash, err := GetHashFromStr(hexString)
	if err != nil {
		panic(err)
	}
	return hash
}

func GetHashFromStr(hashStr string) (hash *Hash, err error) {
	hash = new(Hash)
	bytes, err := GetHashBytesFromStr(hashStr)
	if err != nil {
		return
	}

	copy(hash[:], bytes)

	return
}

func GetHashBytesFromStr(src string) (bytes []byte, err error) {
	if len(src) > MaxHashStringSize {
		return nil, fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)
	}
	var srcBytes []byte
	var srcLen = len(src)
	if srcLen%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+srcLen)
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}
	var reversedHash = make([]byte, Hash256Size)
	_, err = hex.Decode(reversedHash[Hash256Size-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return
	}
	bytes = make([]byte, Hash256Size)
	for i, b := range reversedHash[:Hash256Size/2] {
		bytes[i], bytes[Hash256Size-1-i] = reversedHash[Hash256Size-1-i], b
	}
	return
}

func rotl(x uint64, b uint8) uint64 {
	return (x << b) | (x >> (64 - b))
}

func sipRound(rn []uint64) {
	rn[0] += rn[1]
	rn[1] = rotl(rn[1], 13)
	rn[1] ^= rn[0]
	rn[0] = rotl(rn[0], 32)
	rn[2] += rn[3]
	rn[3] = rotl(rn[3], 16)
	rn[3] ^= rn[2]
	rn[0] += rn[3]
	rn[3] = rotl(rn[3], 21)
	rn[3] ^= rn[0]
	rn[2] += rn[1]
	rn[1] = rotl(rn[1], 17)
	rn[1] ^= rn[2]
	rn[2] = rotl(rn[2], 32)
}

func SipHash(k0, k1 uint64, hash []byte) uint64 {
	d := binary.LittleEndian.Uint64(hash[0:8])
	v0 := uint64(0x736f6d6570736575) ^ k0
	v1 := uint64(0x646f72616e646f6d) ^ k1
	v2 := uint64(0x6c7967656e657261) ^ k0
	v3 := uint64(0x7465646279746573) ^ k1 ^ d

	rn := []uint64{v0, v1, v2, v3}

	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[8:16])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[16:24])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[24:32])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	rn[3] ^= uint64(4) << 59
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= uint64(4) << 59
	rn[2] ^= 0xFF
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)
	return rn[0] ^ rn[1] ^ rn[2] ^ rn[3]
}

func SipHashExtra(k0, k1 uint64, hash []byte, extra uint32) uint64 {
	d := binary.LittleEndian.Uint64(hash[0:8])
	v0 := uint64(0x736f6d6570736575) ^ k0
	v1 := uint64(0x646f72616e646f6d) ^ k1
	v2 := uint64(0x6c7967656e657261) ^ k0
	v3 := uint64(0x7465646279746573) ^ k1 ^ d

	rn := []uint64{v0, v1, v2, v3}

	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[8:16])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[16:24])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = binary.LittleEndian.Uint64(hash[24:32])
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	d = (uint64(36) << 56) | uint64(extra)
	rn[3] ^= d
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= d
	rn[2] ^= 0xFF
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)
	return rn[0] ^ rn[1] ^ rn[2] ^ rn[3]
}

type SipHasher struct {
	v     [4]uint64
	tmp   uint64
	count int
}

func NewSipHasher(k0, k1 uint64) *SipHasher {
	return &SipHasher{
		v: [4]uint64{
			0: uint64(0x736f6d6570736575) ^ k0,
			1: uint64(0x646f72616e646f6d) ^ k1,
			2: uint64(0x6c7967656e657261) ^ k0,
			3: uint64(0x7465646279746573) ^ k1,
		},
	}
}

func (sh *SipHasher) WriteUint64(data uint64) *SipHasher {
	if sh == nil {
		return nil
	}
	if sh.count%8 != 0 {
		panic("expect multiple of 8 bytes have been  written so far")
	}

	rn := make([]uint64, 4)
	copy(rn, sh.v[:])

	rn[3] ^= data
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= data

	copy(sh.v[:], rn)
	sh.count += 8
	return sh
}

func (sh *SipHasher) Write(data []byte) *SipHasher {
	if sh == nil {
		return nil
	}
	t := sh.tmp
	c := sh.count
	size := len(data)

	rn := make([]uint64, 4)
	copy(rn, sh.v[:])

	for i := 0; i < size; i++ {
		t |= uint64(data[i]) << uint(8*(c%8))
		c++
		if c&7 == 0 {
			rn[3] ^= t
			sipRound(rn)
			sipRound(rn)
			rn[0] ^= t
			t = 0
		}
	}
	copy(sh.v[:], rn)
	//sh.v[0] = rn[0]
	//sh.v[1] = rn[1]
	//sh.v[2] = rn[2]
	//sh.v[3] = rn[3]
	sh.count = c
	sh.tmp = t
	return sh
}

func (sh *SipHasher) Finalize() uint64 {
	rn := make([]uint64, 4)
	copy(rn, sh.v[:])

	t := sh.tmp | (uint64(sh.count) << 56)

	rn[3] ^= t
	sipRound(rn)
	sipRound(rn)
	rn[0] ^= t
	rn[2] ^= 0xff
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)
	sipRound(rn)

	return rn[0] ^ rn[1] ^ rn[2] ^ rn[3]
}
