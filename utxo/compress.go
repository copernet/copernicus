package utxo

import (
	"errors"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

const (
	numSpecialScripts = 6
)

func CompressAmount(amt utils.Amount) uint64 {
	n := uint64(amt)
	if n == 0 {
		return 0
	}
	e := uint64(0)
	for n%10 == 0 && e < 9 {
		n /= 10
		e++
	}
	if e < 9 {
		d := uint64(n % 10)
		if d < 1 || d > 9 {
			panic("CompressAmount: d should in range [1,9]")
		}
		n /= 10
		return 1 + (9*n+d-1)*10 + e
	} else {
		return 1 + (n-1)*10 + 9
	}
}

func DecompressAmount(x uint64) utils.Amount {
	if x == 0 {
		return 0
	}
	x--
	e := x % 10
	x /= 10
	n := uint64(0)
	if e < 9 {
		d := (x % 9) + 1
		x /= 9
		n = 10*x + d
	} else {
		n = x + 1
	}
	for e != 0 {
		n *= 10
		e--
	}
	return utils.Amount(n)
}

type scriptCompressor struct {
	script *core.Script
}

func newScriptCompressor(script *core.Script) *scriptCompressor {
	if script == nil {
		return nil
	}
	return &scriptCompressor{
		script: script,
	}
}

func (scr *scriptCompressor) isToKeyId() []byte {
	bs := scr.script.GetScriptByte()
	if len(bs) == 25 && bs[0] == core.OP_DUP && bs[1] == core.OP_HASH160 &&
		bs[2] == 20 && bs[23] == core.OP_EQUALVERIFY && bs[24] == core.OP_CHECKSIG {
		return bs[3:23]
	}
	return nil
}

func (scr *scriptCompressor) isToScriptId() []byte {
	bs := scr.script.GetScriptByte()
	if len(bs) == 23 && bs[0] == core.OP_HASH160 &&
		bs[1] == 20 && bs[22] == core.OP_EQUAL {
		return bs[2:22]
	}
	return nil
}

func (scr *scriptCompressor) isToPubKey() []byte {
	bs := scr.script.GetScriptByte()
	if len(bs) == 35 && bs[0] == 33 && bs[34] == core.OP_CHECKSIG &&
		(bs[1] == 0x2 || bs[1] == 0x3) {
		return bs[1:34]
	}
	if len(bs) == 67 && bs[0] == 65 && bs[66] == core.OP_CHECKSIG &&
		bs[1] == 0x4 {
		if _, err := crypto.ParsePubKey(bs[1:66]); err != nil {
			return nil
		}
		return bs[1:66]
	}
	return nil
}

func (scr *scriptCompressor) Compress() []byte {
	var out []byte
	keyId := scr.isToKeyId()
	if len(keyId) > 0 {
		out = make([]byte, 21)
		out[0] = 0x00
		out = append(out[1:], keyId...)
		return out
	}
	scriptId := scr.isToScriptId()
	if len(scriptId) > 0 {
		out = make([]byte, 21)
		out[0] = 0x01
		out = append(out[1:], scriptId...)
		return out
	}
	pubKey := scr.isToPubKey()
	if len(pubKey) > 0 {
		out = make([]byte, 33)
		copy(out[1:], pubKey[1:33])
		if pubKey[0] == 0x2 || pubKey[0] == 0x3 {
			out[0] = pubKey[0]
			return out
		} else if pubkey[0] == 0x04 {
			out[0] = 0x4 | (pubkey[64] & 0x1)
			return out
		}
	}
	return nil
}

func getSpecialSize(nSize int) int {
	if nSize == 0 || nSize == 1 {
		return 20
	}
	if nSize == 2 || nSize == 3 || nSize == 4 || nSize == 5 {
		return 32
	}
	return 0
}

func (scr *scriptCompressor) Decompress(size int, in []byte) bool {
	var bs []byte
	switch size {
	case 0x00:
		bs = make([]byte, 25)
		bs[0] = core.OP_DUP
		bs[1] = core.OP_HASH160
		bs[2] = 20
		copy(bs[3:], in[0:20])
		bs[23] = core.OP_EQUALVERIFY
		bs[24] = core.OP_CHECKSIG
	case 0x01:
		bs = make([]byte, 23)
		bs[0] = core.OP_HASH160
		bs[1] = 20
		copy(bs[2:], in[0:20])
		bs[22] = core.OP_EQUAL
	case 0x2:
		fallthrough
	case 0x3:
		bs = make([]byte, 35)
		bs[0] = 33
		bs[1] = size
		copy(bs[2:], in[0:32])
		bs[34] = core.OP_CHECKSIG
	case 0x4:
		fallthrough
	case 0x5:
		tmp := make([]byte, 33)
		tmp[0] = size - 2
		copy(tmp[1:], in[0:32])
		if pubkey, err := crypto.ParsePubKey(tmp); err != nil {
			return
		}
		tmp = make([]byte, 65)
		uncompressed := pubkey.SerializeUncompressed(tmp)
		bs = make([]byte, 67)
		bs[0] = 65
		copy(bs[1:], uncompressed)
		bs[66] = core.OP_CHECKSIG
	}
	if bs != nil {
		scr.script = core.NewScriptRaw(bs)
		return true
	}
	return false
}

func (scr *scriptCompressor) Serialize(w io.Writer) error {
	bs := scr.Compress()
	if len(bs) > 0 {
		_, err := w.Write(bs)
		return err
	}
	size := scr.script.Size() + numSpecialScripts
	if err := utils.WriteVarLenInt(w, uint64(size)); err != nil {
		return err
	}
	if _, err := w.Write(scr.script.GetScriptByte()); err != nil {
		return err
	}
	return nil
}

func (scr *scriptCompressor) Unserialize(r io.Reader) error {
	size, err := utils.ReadVarInt(r)
	if err != nil {
		return err
	}
	if size < numSpecialScripts {
		vch := make([]byte, getSpecialSize(size))
		_, err := io.ReadFull(r, vch)
		scr.Decompress(size, vch)
		return err
	}
	size -= numSpecialScripts
	if size > core.MaxScriptSize {
		scr.script.PushOpCode(core.OP_RETURN)
		tmp := make([]byte, size)
		_, err = io.ReadFull(r, tmp)
	} else {
		tmp := make([]byte, size)
		_, err = io.ReadFull(r, tmp)
		if err == nil {
			scr.script = core.NewScriptRaw(tmp)
		}
	}
	return err
}

type TxoutCompressor struct {
	txout *core.TxOut
}

var ErrCompress = errors.New("nil TxoutCompressor receiver")

func NewTxoutCompressor(txout *core.TxOut) *TxoutCompressor {
	if txout == nil {
		return nil
	}
	return &TxoutCompressor{
		txout: txout,
	}
}

func (tc *TxoutCompressor) Serialize(w io.Writer) error {
	if tc == nil {
		return ErrCompress
	}
	amount := CompressAmount(tc.txout.Value)
	if err := utils.WriteVarInt(w, amount); err != nil {
		return err
	}
	sc := newScriptCompressor(tc.txout.Script)
	return sc.Serialize(w)
}

func (tc *TxoutCompressor) Unserialize(r io.Reader) error {
	if tc == nil {
		return ErrCompress
	}
	amount, err := utils.ReadVarInt(r)
	if err != nil {
		return err
	}
	tc.txout.Value = int64(DecompressAmount(amount))
	sc := newScriptCompressor(tc.txout.Script)
	return sc.Unserialize(r)
}
