package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/pow"
)

func CheckBlockHeader(bh * block.BlockHeader,  checkPOW bool) error {
	hash := bh.GetHash()
	params := chain.GetInstance().GetParams()
	if checkPOW{
		flag := new(pow.Pow).CheckProofOfWork(&hash,bh.Bits, params)
		if !flag{
			return  errcode.New(errcode.ErrorPowCheckErr)
		}
	}
	return nil
}