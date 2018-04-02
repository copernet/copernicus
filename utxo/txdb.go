package utxo

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

const (
	DbCoin       byte = 'C'
	DbCoins      byte = 'c'
	DbBlockFiles byte = 'f'
	DbTxIndex    byte = 't'
	DbBlockIndex byte = 'b'

	DbBestBlock   byte = 'B'
	DbFlag        byte = 'F'
	DbReindexFlag byte = 'R'
	DbLastBlock   byte = 'l'
)

func GetTxFromUTXO(hash utils.Hash) *core.Tx {
	return new(core.Tx)
}
