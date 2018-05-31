package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
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

func ContextualCheckBlockHeader(header *block.BlockHeader, preIndex *blockindex.BlockIndex, adjustTime int64) bool {
	nHeight := int32(0)
	if preIndex != nil{
		nHeight = preIndex.Height +1
	}
	gChain := chain.GetInstance()
	params := gChain.GetParams()
	
	p := new(pow.Pow)
	if header.Bits != p.GetNextWorkRequired(preIndex, header, params){
		log.Error("ContextualCheckBlockHeader.GetNextWorkRequired err")
		return false
	}
	blocktime := int64(header.GetBlockTime())
	if blocktime  <= preIndex.GetMedianTimePast(){
		log.Error("ContextualCheckBlockHeader.GetMedianTimePast err")
		return false
	}
	if blocktime > adjustTime + 2*60*60{
		log.Error("ContextualCheckBlockHeader > adjustTime err")
		return false
	}
	if (header.Version < 2 && nHeight >= params.BIP34Height)|| (header.Version<3&&nHeight>=params.BIP66Height) ||(header.Version<4&&nHeight>=params.BIP65Height){
		log.Error("block.version: %d, nheight :%d", header.Version, nHeight)
		return false
	}
	return true
}
