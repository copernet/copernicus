package block

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/pow"
)

func CheckBlockHeader(bh * block.BlockHeader) error {
	hash := bh.GetHash()
	params := chain.GetInstance().GetParams()
	flag := new(pow.Pow).CheckProofOfWork(&hash,bh.Bits, params)
	if !flag{
		err := errcode.New(errcode.ErrorPowCheckErr)
		return err
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
	bi := p.GetNextWorkRequired(preIndex, header, params)
	if header.Bits != bi {
		log.Error("ContextualCheckBlockHeader.GetNextWorkRequired err, preIndexHeight : %d, " +
			"expect bit : %d, actual caculate bit : %d",preIndex.Height, header.Bits, bi )

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
