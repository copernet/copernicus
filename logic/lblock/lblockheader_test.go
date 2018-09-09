package lblock

import (
	"bytes"
	"testing"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
)

// mainNetGenesisHash is the hash of the first block in the block chain for the
// main network (genesis block).
var mainNetGenesisHash = util.Hash([util.Hash256Size]byte{ // Make go vet happy.
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

// mainNetGenesisMerkleRoot is the hash of the first transaction in the genesis
// block for the main network.
var mainNetGenesisMerkleRoot = util.Hash([util.Hash256Size]byte{ // Make go vet happy.
	0x98, 0x20, 0x51, 0xfd, 0x1e, 0x4b, 0xa7, 0x44,
	0xbb, 0xbe, 0x68, 0x0e, 0x1f, 0xee, 0x14, 0x67,
	0x7b, 0xa1, 0xa3, 0xc3, 0x54, 0x0b, 0xf7, 0xb1,
	0xcd, 0xb6, 0x06, 0xe8, 0x57, 0x23, 0x3e, 0x0e,
})

var firstBlockHash = util.Hash([util.Hash256Size]byte{
	0x48, 0x60, 0xeb, 0x18, 0xbf, 0x1b, 0x16, 0x20,
	0xe3, 0x7e, 0x94, 0x90, 0xfc, 0x8a, 0x42, 0x75,
	0x14, 0x41, 0x6f, 0xd7, 0x51, 0x59, 0xab, 0x86,
	0x68, 0x8e, 0x9a, 0x83, 0x00, 0x00, 0x00, 0x00,
})

func TestBlockHeaderGetHash(t *testing.T) {
	blHe := block.NewBlockHeader()
	blHe.Version = 1
	blHe.HashPrevBlock = mainNetGenesisHash
	blHe.MerkleRoot = mainNetGenesisMerkleRoot
	blHe.Time = 1231469665
	blHe.Bits = 0x1d00ffff
	blHe.Nonce = 2573394689

	tmpBlk := block.NewBlockHeader()
	buf := bytes.NewBuffer(nil)
	blHe.Serialize(buf)
	err := tmpBlk.UnserializeHeader(buf)
	if err != nil {
		t.Error("unserialize block header failed.")
	}
	if tmpBlk.Version != blHe.Version {
		t.Errorf("Unserialize late version : %d, expect version : %d", tmpBlk.Version, blHe.Version)
		return
	}
	if !bytes.Equal(tmpBlk.HashPrevBlock[:], blHe.HashPrevBlock[:]) {
		t.Errorf("Unserialize late preHash : %s, expect preHash : %s",
			tmpBlk.HashPrevBlock.String(), blHe.HashPrevBlock.String())
		return
	}
	if !bytes.Equal(tmpBlk.MerkleRoot[:], blHe.MerkleRoot[:]) {
		t.Errorf("Unserialize late merkleRoot : %s, expect merkleRoot : %s",
			tmpBlk.MerkleRoot.String(), blHe.MerkleRoot.String())
		return
	}
	if tmpBlk.Time != blHe.Time {
		t.Errorf("Unserialize late Time : %d, expect Time : %d", tmpBlk.Time, blHe.Time)
		return
	}
	if tmpBlk.Bits != blHe.Bits {
		t.Errorf("Unserialize late bits : %d, expect bits : %d", tmpBlk.Bits, blHe.Bits)
		return
	}
	if tmpBlk.Nonce != blHe.Nonce {
		t.Errorf("Unserialize late Nonce : %d, expect Nonce : %d", tmpBlk.Nonce, blHe.Nonce)
		return
	}
	if blkHash := blHe.GetHash(); blkHash == util.HashZero {
		t.Errorf("cann't get hash")
	} else {
		if !bytes.Equal(blkHash[:], firstBlockHash[:]) {
			t.Errorf("the get block hash is error, actual hash : %s, expect hash : %s\n",
				blkHash.String(), firstBlockHash.String())
		}
	}
}
