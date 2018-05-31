package chainparams

import (
	"time"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/block"
)

var GenesisCoinbaseTx = tx.Tx{}

var GenesisHash = util.Hash([util.Hash256Size]byte{
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

var GenesisMerkleRoot = util.Hash([util.Hash256Size]byte{
	0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
	0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
	0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
	0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a,
})

var GenesisBlock = block.Block{
	Header: block.BlockHeader{
			Version:       1,
			HashPrevBlock: util.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
			MerkleRoot:    util.HexToHash("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
			Time:          uint32(time.Unix(0x495fab29, 0).Nanosecond()), //2009-01-03 18:15:05 +0000 UTC
			Bits:          0x1d00ffff,                                    //486604799  [00000000ffff0000000000000000000000000000000000000000000000000000]
			Nonce:         0x7c2bac1d,                                    // 2083236893
		},
		Txs: []*tx.Tx{&GenesisCoinbaseTx},
}

var RegressionTestGenesisHash = util.Hash([util.Hash256Size]byte{
	0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59,
	0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
	0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f,
	0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
})

var RegressionTestGeneisMerkleRoot = GenesisMerkleRoot

var RegressionTestGenesisBlock = block.Block{
	Header: block.BlockHeader{
			Version:       1,
			HashPrevBlock: util.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
			MerkleRoot:    util.HexToHash("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
			Time:          uint32(time.Unix(1296688602, 0).Nanosecond()), // 2011-02-02 23:16:42 +0000 UTC
			Bits:          0x207fffff,                                    // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
			Nonce:         2,
		},
		Txs: []*tx.Tx{&GenesisCoinbaseTx},
}

var TestNet3GenesisHash = util.Hash([util.Hash256Size]byte{
	0x43, 0x49, 0x7f, 0xd7, 0xf8, 0x26, 0x95, 0x71,
	0x08, 0xf4, 0xa3, 0x0f, 0xd9, 0xce, 0xc3, 0xae,
	0xba, 0x79, 0x97, 0x20, 0x84, 0xe9, 0x0e, 0xad,
	0x01, 0xea, 0x33, 0x09, 0x00, 0x00, 0x00, 0x00,
})

var TestNet3GenesisMerkleRoot = GenesisMerkleRoot

var genesisMerkleRoot = util.Hash([util.Hash256Size]byte{ // Make go vet happy.
	0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2,
	0x7a, 0xc7, 0x2c, 0x3e, 0x67, 0x76, 0x8f, 0x61,
	0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32,
	0x3a, 0x9f, 0xb8, 0xaa, 0x4b, 0x1e, 0x5e, 0x4a,
})

var TestNet3GenesisBlock = block.Block{
		Header: block.BlockHeader{
			Version:       1,
			HashPrevBlock: util.Hash{},
			MerkleRoot:*util.HashFromString("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
			Time:          uint32(time.Unix(1296688602, 0).Unix()), //2011-02-02 23:16:42 +0000 UTC
			Bits:          0x1d00ffff,                                    //486604799  [00000000ffff0000000000000000000000000000000000000000000000000000]
			Nonce:         0x18aea41a,                                    // 414098458
		},
		Txs: []*tx.Tx{&GenesisCoinbaseTx},
}

var SimNetGenesisHash = util.Hash([util.Hash256Size]byte{
	0xf6, 0x7a, 0xd7, 0x69, 0x5d, 0x9b, 0x66, 0x2a,
	0x72, 0xff, 0x3d, 0x8e, 0xdb, 0xbb, 0x2d, 0xe0,
	0xbf, 0xa6, 0x7b, 0x13, 0x97, 0x4b, 0xb9, 0x91,
	0x0d, 0x11, 0x6d, 0x5c, 0xbd, 0x86, 0x3e, 0x68,
})
var SimNetGenesisMerkleRoot = GenesisMerkleRoot
var SimNetGenesisBlock = block.Block{
	Header: block.BlockHeader{
			Version:       1,
			HashPrevBlock: util.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
			MerkleRoot:    util.HexToHash("4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"),
			Time:          uint32(time.Unix(1401292357, 0).Nanosecond()),
			Bits:          0x207fffff,
			Nonce:         2,
		},
	Txs: []*tx.Tx{&GenesisCoinbaseTx},
}

