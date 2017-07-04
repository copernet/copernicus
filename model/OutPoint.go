package model

import (
	"github.com/btccom/copernicus/utils"
	"strconv"
)

type OutPoint struct {
	Hash  *utils.Hash
	Index uint32
}

func NewOutPoint(hash *utils.Hash, index uint32) *OutPoint {
	outPoint := OutPoint{
		Hash:  hash,
		Index: index,
	}
	return &outPoint
}

func (outPoint *OutPoint) String() string {
	// Allocate enough for hash string, colon, and 10 digits.  Although
	// at the time of writing, the number of digits can be no greater than
	// the length of the decimal representation of maxTxOutPerMessage, the
	// maximum message payload may increase in the future and this
	// optimization may go unnoticed, so allocate space for 10 decimal
	// digits, which will fit any uint32.
	buf := make([]byte, 2*utils.HashSize+1, 2*utils.HashSize+1+10)
	copy(buf, outPoint.Hash.ToString())
	buf[2*utils.HashSize] = ':'
	buf = strconv.AppendUint(buf, uint64(outPoint.Index), 10)
	return string(buf)
}
