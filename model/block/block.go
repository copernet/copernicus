package block

import (
	"fmt"


	"github.com/btcboost/copernicus/model/tx"
)

type Block struct {
	Header BlockHeader
	Txs    []*Tx
}