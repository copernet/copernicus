package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/util"
)

const ShortTxIDsLength = 6

type PreFilledTransaction struct {
	Tx    *MsgTx
	Index uint16
}

type MsgCmpctBlock struct {
	shortTxidk0  uint64
	shortTxidk1  uint64
	Nonce        uint64
	ShortTxids   []uint64
	PreFilledTxn []PreFilledTransaction
	Header       block.BlockHeader
}

func NewMsgCmpctBlock(block *MsgBlock) *MsgCmpctBlock {
	nonce, _ := util.RandomUint64()
	shortids := make([]uint64, len(block.Txs)-1)
	prefilledTxn := make([]PreFilledTransaction, 1)
	header := block.Header

	id0, id1, err := fillShortTxIDSelector(&header, nonce)
	if err != nil {
		return nil
	}
	prefilledTxn[0].Index = 0
	prefilledTxn[0].Tx = (*MsgTx)(block.Txs[0])
	for i := 1; i < len(block.Txs); i++ {
		txhash := block.Txs[i].TxHash()
		shortids[i-1] = getShortID(id0, id1, &txhash)
	}
	return &MsgCmpctBlock{
		shortTxidk0:  id0,
		shortTxidk1:  id1,
		Nonce:        nonce,
		ShortTxids:   shortids,
		PreFilledTxn: prefilledTxn,
		Header:       header,
	}
}

func fillShortTxIDSelector(h *block.BlockHeader, nonce uint64) (uint64, uint64, error) {
	bw := bytes.NewBuffer(nil)
	if err := h.Serialize(bw); err != nil {
		return 0, 0, err
	}
	if err := util.WriteElements(bw, nonce); err != nil {
		return 0, 0, err
	}
	hb := util.Sha256Bytes(bw.Bytes())
	return binary.LittleEndian.Uint64(hb[0:8]), binary.LittleEndian.Uint64(hb[8:16]), nil
}

func getShortID(id0, id1 uint64, hash *util.Hash) uint64 {
	return util.SipHash(id0, id1, (*hash)[:]) & 0xffffffffffff
}

func (pft *PreFilledTransaction) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	idx, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	if idx > math.MaxUint16 {
		return messageError("MsgCmpctBlock.Decode", fmt.Sprintf("index overflowed 16-bits"))
	}
	pft.Index = uint16(idx)
	if err := pft.Tx.Decode(r, pver, enc); err != nil {
		return err
	}
	return nil
}

func (pft *PreFilledTransaction) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if err := util.WriteVarInt(w, uint64(pft.Index)); err != nil {
		return err
	}
	if err := pft.Tx.Encode(w, pver, enc); err != nil {
		return err
	}
	return nil
}

func (msg *MsgCmpctBlock) Decode(r io.Reader, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("cmpctblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgCmpctBlock.Decode", str)
	}
	if err := msg.Header.Unserialize(r); err != nil {
		return err
	}
	if err := util.ReadElements(r, &msg.Nonce); err != nil {
		return err
	}
	shortIDSize, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	ids := make([]uint64, shortIDSize)
	for i := 0; i < len(ids); i++ {
		lsb := uint32(0)
		msb := uint16(0)
		if err := util.ReadElements(r, &lsb); err != nil {
			return err
		}
		if err := util.ReadElements(r, &msb); err != nil {
			return err
		}
		ids[i] = (uint64(msb) << 32) | uint64(lsb)
	}
	msg.ShortTxids = append(msg.ShortTxids, ids...)
	pftLen, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	vpft := make([]PreFilledTransaction, pftLen)
	for i := 0; i < len(vpft); i++ {
		if err := vpft[i].Decode(r, pver, enc); err != nil {
			return err
		}
	}
	id0, id1, err := fillShortTxIDSelector(&msg.Header, msg.Nonce)
	if err != nil {
		return err
	}
	msg.shortTxidk0 = id0
	msg.shortTxidk1 = id1
	return nil
}

func (msg *MsgCmpctBlock) Encode(w io.Writer, pver uint32, enc MessageEncoding) error {
	if pver < ShortIdsBlocksVersion {
		str := fmt.Sprintf("cmpctblock message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgCmpctBlock.Encode", str)
	}
	if err := msg.Header.Serialize(w); err != nil {
		return err
	}
	if err := util.WriteElements(w, &msg.Nonce); err != nil {
		return err
	}
	if err := util.WriteVarInt(w, uint64(len(msg.ShortTxids))); err != nil {
		return err
	}
	for i := 0; i < len(msg.ShortTxids); i++ {
		lsb := uint32(0)
		msb := uint16(0)
		lsb = uint32(msg.ShortTxids[i] & 0xffffffff)
		msb = uint16((msg.ShortTxids[i] >> 32) & 0xffff)
		if err := util.WriteElements(w, lsb); err != nil {
			return err
		}
		if err := util.WriteElements(w, msb); err != nil {
			return err
		}
	}
	for i := 0; i < len(msg.PreFilledTxn); i++ {
		if err := msg.PreFilledTxn[i].Encode(w, pver, enc); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgCmpctBlock) Command() string {
	return CmdCmpctBlock
}

func (msg *MsgCmpctBlock) MaxPayloadLength(pver uint32) uint32 {
	return uint32(80 + 8 + 3 + len(msg.ShortTxids)*6 + 3 + len(msg.PreFilledTxn)*(3+MaxBlockPayload))
}
