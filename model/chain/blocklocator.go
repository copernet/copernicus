package chain

import (
	"github.com/copernet/copernicus/util"
)

type BlockLocator struct {
	blockHashList []util.Hash
}

func NewBlockLocator(vHaveIn []util.Hash) *BlockLocator {
	blo := BlockLocator{}
	blo.blockHashList = vHaveIn
	return &blo
}

func (blt *BlockLocator) SetNull() {
	blt.blockHashList = make([]util.Hash, 0)
}

func (blt *BlockLocator) IsNull() bool {
	return len(blt.blockHashList) == 0
}

func (blt *BlockLocator) GetBlockHashList() []util.Hash {
	return blt.blockHashList
}
