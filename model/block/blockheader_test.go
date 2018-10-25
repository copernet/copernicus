package block

import (
	"bytes"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var blockOne = []byte{
	0x01, 0x00, 0x00, 0x00, // Version 1
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00, // PrevBlock
	0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
	0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
	0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
	0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e, // MerkleRoot
	0x61, 0xbc, 0x66, 0x49, // Timestamp
	0xff, 0xff, 0x00, 0x1d, // Bits
	0x01, 0xe3, 0x62, 0x99, // Nonce
}

var blockTenThousand = []byte{
	0x01, 0x00, 0x00, 0x00, // Version 1
	0x50, 0x12, 0x01, 0x19, 0x17, 0x2a, 0x61, 0x04,
	0x21, 0xa6, 0xc3, 0x01, 0x1d, 0xd3, 0x30, 0xd9,
	0xdf, 0x07, 0xb6, 0x36, 0x16, 0xc2, 0xcc, 0x1f,
	0x1c, 0xd0, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // PrevBlock
	0x66, 0x57, 0xa9, 0x25, 0x2a, 0xac, 0xd5, 0xc0,
	0xb2, 0x94, 0x09, 0x96, 0xec, 0xff, 0x95, 0x22,
	0x28, 0xc3, 0x06, 0x7c, 0xc3, 0x8d, 0x48, 0x85,
	0xef, 0xb5, 0xa4, 0xac, 0x42, 0x47, 0xe9, 0xf3, // MerkleRoot
	0x37, 0x22, 0x1b, 0x4d, // Timestamp
	0x4c, 0x86, 0x04, 0x1b, // Bits
	0x0f, 0x2b, 0x57, 0x10, // Nonce
}

var testBlocks = [][]byte{
	blockOne,
	blockTenThousand,
}

func TestHeadersWire(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	hdr := NewBlockHeader()
	for i, blk := range testBlocks {
		buf.Reset()
		buf.Write(blk)
		if err := hdr.Decode(buf); err != nil {
			t.Errorf("test %d , Decode failed :%v", i, err)
		}
		if err := hdr.Encode(buf); err != nil {
			t.Errorf("test %d , Encode failed :%v", i, err)
		}
	}
}

func TestBlockHeaderDecodeEncode(t *testing.T) {
	hdr := NewBlockHeader()
	buf := bytes.NewBuffer(nil)
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		hdr2 := NewBlockHeader()
		hdr.SetNull()
		hdr.Version = r.Int31()
		hdr.Time = r.Uint32()
		hdr.Bits = r.Uint32()
		hdr.Nonce = r.Uint32()
		for j := 0; j < util.Hash256Size; j++ {
			hdr.MerkleRoot[j] = byte(r.Intn(256))
			hdr.HashPrevBlock[j] = byte(r.Intn(256))
		}
		assert.Equal(t, blockHeaderLength, hdr.SerializeSize())
		assert.Equal(t, blockHeaderLength, hdr.EncodeSize())

		buf.Reset()
		err := hdr.Encode(buf)
		if err != nil {
			t.Error("block header encode failed.")
		}
		err = hdr2.Decode(buf)
		if err != nil {
			t.Error("block header decode failed.")
		}

		// check serialize size
		assert.Equal(t, hdr.SerializeSize(), hdr2.SerializeSize())
		assert.Equal(t, hdr.EncodeSize(), hdr2.EncodeSize())

		// check block header hash
		assert.Equal(t, hdr.GetHash(), hdr2.GetHash())
		assert.Equal(t, hdr.String(), hdr2.String())
	}
}

func TestBlockHeaderNull(t *testing.T) {
	hdr := NewBlockHeader()
	assert.True(t, hdr.IsNull())

	r := rand.New(rand.NewSource(time.Now().Unix()))
	hdr.Version = r.Int31()
	hdr.Time = r.Uint32()
	hdr.Bits = r.Uint32()
	hdr.Nonce = r.Uint32()
	for j := 0; j < util.Hash256Size; j++ {
		hdr.MerkleRoot[j] = byte(r.Intn(256))
		hdr.HashPrevBlock[j] = byte(r.Intn(256))
	}
	assert.False(t, hdr.IsNull())

	hdr.SetNull()
	assert.True(t, hdr.IsNull())
}

func TestGetHeaderSerializeList(t *testing.T) {
	hdr := NewBlockHeader()
	expects := []string{"Version", "HashPrevBlock", "MerkleRoot", "Time", "Bits", "Nonce"}
	dumplist := hdr.GetSerializeList()
	assert.Equal(t, expects, dumplist)
}
