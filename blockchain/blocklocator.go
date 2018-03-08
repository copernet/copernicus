package blockchain

import (
	"io"

	"github.com/btcboost/copernicus/utils"
)

type BlockLocator struct {
	vHave []utils.Hash
}

func NewBlockLocator(vHaveIn []utils.Hash) *BlockLocator {
	blo := BlockLocator{}
	blo.vHave = vHaveIn
	return &blo
}

func SerializationOp(w io.Writer) {

}
