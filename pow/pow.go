package pow

import (
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/utils"
)

type Pow struct{}

func (pow *Pow) GetNextWorkRequired(pindexPrev *blockchain.BlockIndex, blHeader *model.BlockHeader, params *msg.BitcoinParams) uint32 {
	if pindexPrev == nil {
		return 0
	}

	return 0
}

func (pow *Pow) CalculateNextWorkRequired(pindexPrev *blockchain.BlockIndex, firstBlockTime int64, params *msg.BitcoinParams) uint32 {
	return 0
}

func (pow *Pow) CheckProofOfWork(hash utils.Hash, bits uint32, params *msg.BitcoinParams) bool {
	return true
}

func (pow *Pow) GetNextCashWorkRequired(pindexPrev *blockchain.BlockIndex, blHeader *model.BlockHeader, params *msg.BitcoinParams) uint32 {
	return 0
}
