package lblock

import (
	"fmt"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/pow"
)

func CheckBlockHeader(bh *block.BlockHeader) error {
	hash := bh.GetHash()
	params := chain.GetInstance().GetParams()
	flag := new(pow.Pow).CheckProofOfWork(&hash, bh.Bits, params)
	if !flag {
		log.Error("CheckBlockHeader CheckProofOfWork err")
		err := errcode.NewError(errcode.RejectInvalid, "high-hash")
		return err
	}
	return nil
}

func ContextualCheckBlockHeader(header *block.BlockHeader, preIndex *blockindex.BlockIndex, adjustTime int64) (err error) {
	nHeight := int32(0)
	if preIndex != nil {
		nHeight = preIndex.Height + 1
	}
	gChain := chain.GetInstance()
	params := gChain.GetParams()

	p := new(pow.Pow)
	if header.Bits != p.GetNextWorkRequired(preIndex, header, params) {
		log.Error("ContextualCheckBlockHeader.GetNextWorkRequired err")
		return errcode.NewError(errcode.RejectInvalid, "bad-diffbits")
	}

	blocktime := int64(header.Time)
	if blocktime <= preIndex.GetMedianTimePast() {
		log.Error("ContextualCheckBlockHeader: block's timestamp is too early")
		return errcode.NewError(errcode.RejectInvalid, "time-too-old")
	}

	if blocktime > adjustTime+2*60*60 {
		log.Error("ContextualCheckBlockHeader: block's timestamp far in the future. block time:%d, adjust time:%d", blocktime, adjustTime)
		return errcode.NewError(errcode.RejectInvalid, "time-too-new")
	}

	if (header.Version < 2 && nHeight >= params.BIP34Height) || (header.Version < 3 && nHeight >= params.BIP66Height) || (header.Version < 4 && nHeight >= params.BIP65Height) {
		log.Error("block.version: %d, nheight :%d", header.Version, nHeight)
		return errcode.NewError(errcode.RejectObsolete, fmt.Sprintf("bad-version(0x%08x)", header.Version))
	}

	return nil
}
