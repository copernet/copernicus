package txout

import (
	//"encoding/hex"
	"errors"
	//"fmt"
	"io"
	//"os"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

const (
	numSpecialScripts = 6
)

func CompressAmount(amt amount.Amount) uint64 {
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
		d := n % 10
		if d < 1 || d > 9 {
			panic("CompressAmount: d should in range [1,9]")
		}
		n /= 10
		return 1 + (9*n+d-1)*10 + e
	}
	return 1 + (n-1)*10 + 9
}

func DecompressAmount(x uint64) amount.Amount {
	if x == 0 {
		return 0
	}
	x--
	e := x % 10
	x /= 10
	var n uint64
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
	return amount.Amount(n)
}

type scriptCompressor struct {
	sp **script.Script
}

func newScriptCompressor(sp **script.Script) *scriptCompressor {
	if sp == nil {
		return nil
	}
	if *sp == nil {
		*sp = script.NewEmptyScript()
	}
	return &scriptCompressor{
		sp: sp,
	}
}

func (scr *scriptCompressor) isToKeyID() []byte {
	so := *scr.sp
	bs := so.GetData()
	if len(bs) == 25 && bs[0] == opcodes.OP_DUP && bs[1] == opcodes.OP_HASH160 &&
		bs[2] == 20 && bs[23] == opcodes.OP_EQUALVERIFY && bs[24] == opcodes.OP_CHECKSIG {
		return bs[3:23]
	}
	return nil
}

func (scr *scriptCompressor) isToScriptID() []byte {
	so := *scr.sp
	bs := so.GetData()
	if len(bs) == 23 && bs[0] == opcodes.OP_HASH160 &&
		bs[1] == 20 && bs[22] == opcodes.OP_EQUAL {
		return bs[2:22]
	}
	return nil
}

func (scr *scriptCompressor) isToPubKey() []byte {
	so := *scr.sp
	bs := so.GetData()
	if len(bs) == 35 && bs[0] == 33 && bs[34] == opcodes.OP_CHECKSIG &&
		(bs[1] == 0x2 || bs[1] == 0x3) {
		return bs[1:34]
	}
	if len(bs) == 67 && bs[0] == 65 && bs[66] == opcodes.OP_CHECKSIG &&
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
	keyID := scr.isToKeyID()
	if len(keyID) > 0 {
		out = make([]byte, 21)
		out[0] = 0x00
		copy(out[1:], keyID)
		return out
	}
	scriptID := scr.isToScriptID()
	if len(scriptID) > 0 {
		out = make([]byte, 21)
		out[0] = 0x01
		copy(out[1:], scriptID)
		return out
	}
	pubKey := scr.isToPubKey()
	if len(pubKey) > 0 {
		out = make([]byte, 33)
		copy(out[1:], pubKey[1:33])
		if pubKey[0] == 0x2 || pubKey[0] == 0x3 {
			out[0] = pubKey[0]
			return out
		} else if pubKey[0] == 0x04 {
			out[0] = 0x4 | (pubKey[64] & 0x1)
			return out
		}
	}
	return nil
}

func getSpecialSize(nSize uint64) int {
	if nSize == 0 || nSize == 1 {
		return 20
	}
	if nSize == 2 || nSize == 3 || nSize == 4 || nSize == 5 {
		return 32
	}
	return 0
}

func (scr *scriptCompressor) Decompress(size uint64, in []byte) bool {
	var bs []byte
	switch size {
	case 0x00:
		bs = make([]byte, 25)
		bs[0] = opcodes.OP_DUP
		bs[1] = opcodes.OP_HASH160
		bs[2] = 20
		copy(bs[3:], in[0:20])
		bs[23] = opcodes.OP_EQUALVERIFY
		bs[24] = opcodes.OP_CHECKSIG
		//fmt.Fprintf(os.Stderr, "after case 0x00,bs=%s\n", hex.EncodeToString(bs))
	case 0x01:
		bs = make([]byte, 23)
		bs[0] = opcodes.OP_HASH160
		bs[1] = 20
		copy(bs[2:], in[0:20])
		bs[22] = opcodes.OP_EQUAL
	case 0x2:
		fallthrough
	case 0x3:
		bs = make([]byte, 35)
		bs[0] = 33
		bs[1] = byte(size)
		copy(bs[2:], in[0:32])
		bs[34] = opcodes.OP_CHECKSIG
	case 0x4:
		fallthrough
	case 0x5:
		tmp := make([]byte, 33)
		tmp[0] = byte(size - 2)
		copy(tmp[1:], in[0:32])
		pubkey, err := crypto.ParsePubKey(tmp)
		if err != nil {
			return false
		}
		uncompressed := pubkey.SerializeUncompressed()
		bs = make([]byte, 67)
		bs[0] = 65
		copy(bs[1:], uncompressed)
		bs[66] = opcodes.OP_CHECKSIG
	}
	if bs != nil {
		*scr.sp = script.NewScriptRaw(bs)
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
	so := *scr.sp
	size := so.Size() + numSpecialScripts
	if err := util.WriteVarLenInt(w, uint64(size)); err != nil {
		return err
	}
	if _, err := w.Write(so.GetData()); err != nil {
		return err
	}
	return nil
}

func (scr *scriptCompressor) Unserialize(r io.Reader) error {
	size, err := util.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	//fmt.Fprintf(os.Stderr, "got size=%d\n", size)
	if size < numSpecialScripts {
		vch := make([]byte, getSpecialSize(size))
		if _, err := io.ReadFull(r, vch); err != nil {
			//fmt.Fprintf(os.Stderr, "io.ReadFull=%v\n", err)
			return err
		}
		//fmt.Fprintf(os.Stderr, "got vch=%s\n", hex.EncodeToString(vch))
		if !scr.Decompress(size, vch) {
			return errors.New("scriptCompressor.Decompress: return false")
		}
		return nil
	}
	size -= numSpecialScripts
	if size > script.MaxScriptSize {
		(*scr.sp).PushOpCode(opcodes.OP_RETURN)
		tmp := make([]byte, size)
		_, err = io.ReadFull(r, tmp)
	} else {
		tmp := make([]byte, size)
		_, err = io.ReadFull(r, tmp)
		//fmt.Fprintf(os.Stderr, "after readfull tmp=%s, err=%v\n", hex.EncodeToString(tmp), err)
		if err == nil {
			*scr.sp = script.NewScriptRaw(tmp)
		}
	}
	return err
}

type TxoutCompressor struct {
	txout *TxOut
	sc    *scriptCompressor
}

var ErrCompress = errors.New("nil TxoutCompressor receiver")

func NewTxoutCompressor(txout *TxOut) *TxoutCompressor {
	if txout == nil {
		return nil
	}
	return &TxoutCompressor{
		txout: txout,
		sc:    newScriptCompressor(&txout.scriptPubKey),
	}
}

func (tc *TxoutCompressor) Serialize(w io.Writer) error {
	if tc == nil {
		return ErrCompress
	}
	count := CompressAmount(tc.txout.value)
	if err := util.WriteVarLenInt(w, count); err != nil {
		return err
	}
	return tc.sc.Serialize(w)
}

func (tc *TxoutCompressor) Unserialize(r io.Reader) error {
	if tc == nil {
		return ErrCompress
	}
	count, err := util.ReadVarLenInt(r)
	if err != nil {
		log.Error("TxoutCompressor Unserialize: the read count is: %d, error: %v", count, err)
		return err
	}
	tc.txout.value = DecompressAmount(count)
	return tc.sc.Unserialize(r)
}
