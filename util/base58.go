package util

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
)

const (
	alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

var (
	// ErrBadChecksum represents the checksum of data is not right
	ErrBadChecksum = errors.New("invalid format: bad checksum")
	// ErrInvalidFormat represents the encoded string format is not right
	ErrInvalidFormat = errors.New("invalid format: version and/or checksum bytes missing")
	// ErrInvalidCharacter represents the character of string is invalid
	ErrInvalidCharacter = errors.New("invalid character")
)

var (
	big58               = big.NewInt(58)
	big0                = big.NewInt(0)
	alphabetLookupTable = [256]byte{
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 0, 1, 2, 3, 4, 5, 6,
		7, 8, 255, 255, 255, 255, 255, 255,
		255, 9, 10, 11, 12, 13, 14, 15,
		16, 255, 17, 18, 19, 20, 21, 255,
		22, 23, 24, 25, 26, 27, 28, 29,
		30, 31, 32, 255, 255, 255, 255, 255,
		255, 33, 34, 35, 36, 37, 38, 39,
		40, 41, 42, 43, 255, 44, 45, 46,
		47, 48, 49, 50, 51, 52, 53, 54,
		55, 56, 57, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
	}
)

func checksum(data []byte) []byte {
	hash := sha256.Sum256(data)
	hash = sha256.Sum256(hash[:])
	return hash[:4]
}

// Base58EncodeCheck is used for encoding Bitcoin addresses
// see https://en.bitcoin.it/wiki/Base58Check_encoding
func Base58EncodeCheck(data []byte, version byte) string {
	b := make([]byte, 0, 1+len(data)+4)
	b = append(append(b, version), data...)
	return Base58Encode(append(b, checksum(b)...))
}

// Base58DecodeCheck is used for decoding Bitcoin addresses
// see https://en.bitcoin.it/wiki/Base58Check_encoding
func Base58DecodeCheck(str string) ([]byte, byte, error) {
	data, err := Base58Decode(str)
	if err != nil {
		return nil, 0, err
	}

	ndata := len(data)

	// version + data + checksum
	if ndata < 5 {
		return nil, 0, ErrInvalidFormat
	}

	if !bytes.Equal(checksum(data[:ndata-4]), data[ndata-4:]) {
		return nil, 0, ErrBadChecksum
	}

	return data[1 : ndata-4], data[0], nil
}

// Base58Encode represents base58 encode method
func Base58Encode(data []byte) string {
	return trezorBase58Encode(data)
}

// Base58Decode represents base58 decode method
func Base58Decode(str string) ([]byte, error) {
	return trezorBase58Decode(str)
}

// bigintBase58Encode use big.Int to implement base58 encode
func bigintBase58Encode(data []byte) string {
	b := new(big.Int).SetBytes(data)

	rv := make([]byte, 0, len(data)*137/100)

	for b.Cmp(big0) > 0 {
		mod := new(big.Int)
		b.DivMod(b, big58, mod)
		rv = append(rv, alphabet[mod.Int64()])
	}

	for _, v := range data {
		if v != 0 {
			break
		}
		rv = append(rv, alphabet[0])
	}

	n := len(rv)
	for i := 0; i < n/2; i++ {
		rv[i], rv[n-1-i] = rv[n-1-i], rv[i]
	}

	return string(rv)
}

// bigintBase58Encode use big.Int to implement base58 decode
func bigintBase58Decode(str string) ([]byte, error) {
	rv := big.NewInt(0)
	nth := big.NewInt(1)
	right := big.NewInt(1)

	for i := len(str) - 1; i >= 0; i-- {
		base := alphabetLookupTable[str[i]]
		if base == 255 {
			return nil, ErrInvalidCharacter
		}

		// rv = rv + 58 ^ nth * base
		right.SetInt64(int64(base))
		right.Mul(nth, right)
		rv.Add(rv, right)
		nth.Mul(nth, big58)
	}

	zcount := 0
	for ; zcount < len(str) && str[zcount] == alphabet[0]; zcount++ {
	}

	bytes := rv.Bytes()
	nbytes := len(bytes)
	data := make([]byte, zcount+nbytes)

	for i := 0; i < zcount+nbytes; i++ {
		if i < zcount {
			data[i] = 0
		} else {
			data[i] = bytes[i-zcount]
		}
	}

	return data, nil
}

// trezorBase58Encode use fast algorithm to implement base58 encode
// see https://github.com/trezor/trezor-crypto/blob/master/base58.c#L152
func trezorBase58Encode(data []byte) string {
	ndata := len(data)
	zcount := 0

	for zcount < ndata && data[zcount] == 0 {
		zcount++
	}

	// log(256,10)/log(58,10)
	size := (ndata-zcount)*137/100 + 1
	high := size - 1
	j := size - 1
	i := zcount
	buf := make([]byte, size)

	for ; i < ndata; i++ {
		carry := uint32(data[i])

		for j = size - 1; j > high || carry != 0; j-- {
			carry += 256 * uint32(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}

		high = j
	}

	for j = 0; j < size && buf[j] == 0; j++ {
	}

	b58 := make([]byte, size-j+zcount)

	for i = 0; i < zcount; i++ {
		b58[i] = alphabet[0]
	}

	for i = zcount; j < size; i++ {
		b58[i] = alphabet[buf[j]]
		j++
	}

	return string(b58)
}

// trezorBase58Encode use fast algorithm to implement base58 decode
// see https://github.com/trezor/trezor-crypto/blob/master/base58.c#L152
func trezorBase58Decode(str string) ([]byte, error) {
	nstr := len(str)
	zcount := 0

	for zcount < nstr && str[zcount] == alphabet[0] {
		zcount++
	}

	// log(58, 10)/log(256, 10)
	size := (nstr-zcount)*733/1000 + 1
	high := size - 1
	j := size - 1
	i := zcount
	buf := make([]byte, size)

	for ; i < nstr; i++ {
		carry := uint32(alphabetLookupTable[str[i]])
		if carry == 255 {
			return nil, ErrInvalidCharacter
		}

		for j = size - 1; j > high || carry != 0; j-- {
			carry += 58 * uint32(buf[j])
			buf[j] = byte(carry % 256)
			carry /= 256
		}

		high = j
	}

	for j = 0; j < size && buf[j] == 0; j++ {
	}

	b256 := make([]byte, size-j+zcount)

	for i = 0; i < zcount; i++ {
		b256[i] = 0
	}

	for i = zcount; j < size; i++ {
		b256[i] = buf[j]
		j++
	}

	return b256, nil
}
