package script

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/util"
	"github.com/pkg/errors"
)

const (
	MaxMessagePayload = 32 * 1024 * 1024
)

const (
	DefaultSize = 28

	// MaxPubKeysPerMultiSig :  maximum number of public keys per multiSig
	MaxPubKeysPerMultiSig = 20

	// LockTimeThreshold threshold for nLockTime: below this value it is interpreted as block number,
	// otherwise as UNIX timestamp. Threshold is Tue Nov 5 00:53:20 1985 UTC
	LockTimeThreshold = 500000000

	// SequenceFinal setting sequence to this value for every input in a transaction
	// disables nLockTime.
	SequenceFinal = 0xffffffff

	MaxScriptSize        = 10000
	MaxScriptElementSize = 520
	MaxScriptOpCodes     = 201
	MaxOpsPerScript      = 201
)

const (
	// SequenceLockTimeDisableFlag below flags apply in the context of BIP 68*/
	// If this flag set, CTxIn::nSequence is NOT interpreted as a
	// relative lock-time. */
	SequenceLockTimeDisableFlag = 1 << 31

	// SequenceLockTimeTypeFlag if CTxIn::nSequence encodes a relative lock-time and this flag
	// is set, the relative lock-time has units of 512 seconds,
	// otherwise it specifies blocks with a granularity of 1.
	SequenceLockTimeTypeFlag = 1 << 22

	// SequenceLockTimeMask if CTxIn::nSequence encodes a relative lock-time, this mask is
	// applied to extract that lock-time from the sequence field.
	SequenceLockTimeMask = 0x0000ffff

	// SequenceLockTimeGranularity in order to use the same number of bits to encode roughly the
	// same wall-clock duration, and because blocks are naturally
	// limited to occur every 600s on average, the minimum granularity
	// for time-based relative lock-time is fixed at 512 seconds.
	// Converting from CTxIn::nSequence to seconds is performed by
	// multiplying by 512 = 2^9, or equivalently shifting up by
	// 9 bits.
	SequenceLockTimeGranularity = 9
)

/** Script verification flags */
const (
	ScriptVerifyNone = 0

	// Evaluate P2SH subscripts (softfork safe, BIP16).
	ScriptVerifyP2SH = (1 << 0)

	// Passing a non-strict-DER signature or one with undefined hashtype to a
	// checksig operation causes script failure. Evaluating a pubkey that is not
	// (0x04 + 64 bytes) or (0x02 or 0x03 + 32 bytes) by checksig causes script
	// failure.
	ScriptVerifyStrictEnc = (1 << 1)

	// Passing a non-strict-DER signature to a checksig operation causes script
	// failure (softfork safe, BIP62 rule 1)
	ScriptVerifyDersig = (1 << 2)

	// Passing a non-strict-DER signature or one with S > order/2 to a checksig
	// operation causes script failure
	// (softfork safe, BIP62 rule 5).
	ScriptVerifyLowS = (1 << 3)

	// verify dummy stack item consumed by CHECKMULTISIG is of zero-length
	// (softfork safe, BIP62 rule 7).
	ScriptVerifyNullDummy = (1 << 4)

	// Using a non-push operator in the scriptSig causes script failure
	// (softfork safe, BIP62 rule 2).
	ScriptVerifySigPushOnly = (1 << 5)

	// Require minimal encodings for all push operations (OP_0... OP_16,
	// OP_1NEGATE where possible, direct pushes up to 75 bytes, OP_PUSHDATA up
	// to 255 bytes, OP_PUSHDATA2 for anything larger). Evaluating any other
	// push causes the script to fail (BIP62 rule 3). In addition, whenever a
	// stack element is interpreted as a number, it must be of minimal length
	// (BIP62 rule 4).
	// (softfork safe)
	ScriptVerifyMinmalData = (1 << 6)

	// Discourage use of Nops reserved for upgrades (NOP1-10)
	//
	// Provided so that nodes can avoid accepting or mining transactions
	// containing executed NOP's whose meaning may change after a soft-fork,
	// thus rendering the script invalid; with this flag set executing
	// discouraged NOPs fails the script. This verification flag will never be a
	// mandatory flag applied to scripts in a block. NOPs that are not executed,
	// e.g.  within an unexecuted IF ENDIF block, are *not* rejected.
	ScriptVerifyDiscourageUpgradableNops = (1 << 7)

	// Require that only a single stack element remains after evaluation. This
	// changes the success criterion from "At least one stack element must
	// remain, and when interpreted as a boolean, it must be true" to "Exactly
	// one stack element must remain, and when interpreted as a boolean, it must
	// be true".
	// (softfork safe, BIP62 rule 6)
	// Note: CLEANSTACK should never be used without P2SH or WITNESS.
	ScriptVerifyCleanStack = (1 << 8)

	// Verify CHECKLOCKTIMEVERIFY
	//
	// See BIP65 for details.
	ScriptVerifyCheckLockTimeVerify = (1 << 9)

	// support CHECKSEQUENCEVERIFY opcode
	//
	// See BIP112 for details
	ScriptVerifyCheckSequenceVerify = (1 << 10)

	// Making v1-v16 witness program non-standard
	//
	ScriptVerifyDiscourageUpgradableWitnessProgram = (1 << 12)

	// Segwit script only: Require the argument of OP_IF/NOTIF to be exactly
	// 0x01 or empty vector
	//
	ScriptVerifyMinimalIf = (1 << 13)

	// Signature(s) must be empty vector if an CHECK(MULTI)SIG operation failed
	//
	ScriptVerifyNullFail = (1 << 14)

	// Public keys in scripts must be compressed
	//
	ScriptVerifyCompressedPubkeyType = (1 << 15)

	// ScriptEnableSigHashForkID Do we accept signature using SIGHASH_FORKID
	//
	ScriptEnableSigHashForkID = (1 << 16)

	// Do we accept activate replay protection using a different fork id.
	//
	ScriptEnableReplayProtection = (1 << 17)

	// Enable new opcodes.
	//
	ScriptEnableMonolithOpcodes = (1 << 18)
)

const (
	ScriptNonStandard = iota
	// 'standard' transaction types:
	ScriptPubkey
	ScriptPubkeyHash
	ScriptHash
	ScriptMultiSig
	ScriptNullData

	MaxOpReturnRelay      uint = 83
	MaxOpReturnRelayLarge uint = 223
)

const (
	// MandatoryScriptVerifyFlags mandatory script verification flags that all new blocks must comply with for
	// them to be valid. (but old blocks may not comply with) Currently just P2SH,
	// but in the future other flags may be added, such as a soft-fork to enforce
	// strict DER encoding.
	//
	// Failing one of these tests may trigger a DoS ban - see CheckInputs() for
	// details.
	MandatoryScriptVerifyFlags uint = ScriptVerifyP2SH | ScriptVerifyStrictEnc | ScriptEnableSigHashForkID

	/*StandardScriptVerifyFlags standard script verification flags that standard transactions will comply
	 * with. However scripts violating these flags may still be present in valid
	 * blocks and we must accept those blocks.
	 */
	StandardScriptVerifyFlags uint = MandatoryScriptVerifyFlags | ScriptVerifyDersig |
		ScriptVerifyMinmalData | ScriptVerifyNullDummy |
		ScriptVerifyDiscourageUpgradableNops | ScriptVerifyCleanStack |
		ScriptVerifyNullFail | ScriptVerifyCheckLockTimeVerify |
		ScriptVerifyCheckSequenceVerify | ScriptVerifyLowS |
		ScriptVerifyDiscourageUpgradableWitnessProgram

	/*StandardNotMandatoryVerifyFlags for convenience, standard but not mandatory verify flags. */
	StandardNotMandatoryVerifyFlags uint = StandardScriptVerifyFlags & (^MandatoryScriptVerifyFlags)
)

type Script struct {
	data          []byte
	ParsedOpCodes []opcodes.ParsedOpCode
	badOpCode     bool
}

func (s *Script) SerializeSize() uint32 {
	return s.EncodeSize()
}

func (s *Script) Serialize(writer io.Writer) (err error) {
	return s.Encode(writer)
}

func (s *Script) Unserialize(reader io.Reader, isCoinBase bool) (err error) {
	return s.Decode(reader, isCoinBase)
}

func (s *Script) EncodeSize() uint32 {
	return util.VarIntSerializeSize(uint64(len(s.data))) + uint32(len(s.data))
}

func (s *Script) Encode(writer io.Writer) (err error) {
	//log.Debug("script data %s", hex.EncodeToString(s.data))
	return util.WriteVarBytes(writer, s.data)
}

func (s *Script) Decode(reader io.Reader, isCoinBase bool) (err error) {
	bytes, err := ReadScript(reader, MaxMessagePayload, "tx input signature script")
	if err != nil {
		return err
	}
	s.data = bytes
	if isCoinBase {
		return nil
	}
	err = s.convertOPS()
	if err != nil {
		return err
	}

	return nil
}

func (s *Script) IsSpendable() bool {
	if (len(s.data) > 0 && s.data[0] == opcodes.OP_RETURN) || len(s.data) > MaxScriptSize {
		return false
	}
	return true
}

func NewScriptRaw(bytes []byte) *Script {
	newBytes := make([]byte, len(bytes))
	copy(newBytes, bytes)
	s := Script{data: newBytes}
	//convertOPS maybe failed, but it doesn't matter
	if s.convertOPS() != nil {
		s.badOpCode = true
	}
	return &s
}

func NewScriptOps(oldParsedOpCodes []opcodes.ParsedOpCode) *Script {
	newParsedOpCodes := make([]opcodes.ParsedOpCode, 0, len(oldParsedOpCodes))
	for _, oldParsedOpCode := range oldParsedOpCodes {
		newParsedOpCodes = append(newParsedOpCodes, *opcodes.NewParsedOpCode(oldParsedOpCode.OpValue,
			oldParsedOpCode.Length, oldParsedOpCode.Data))
	}
	s := Script{ParsedOpCodes: newParsedOpCodes}
	s.convertRaw()
	s.badOpCode = false
	return &s
}

func NewEmptyScript() *Script {
	s := Script{}
	s.data = make([]byte, 0)
	s.ParsedOpCodes = make([]opcodes.ParsedOpCode, 0)
	s.badOpCode = false
	return &s
}

func (s *Script) convertRaw() {
	s.data = make([]byte, 0)
	for _, e := range s.ParsedOpCodes {
		s.data = append(s.data, e.OpValue)
		if e.OpValue == opcodes.OP_PUSHDATA1 {
			s.data = append(s.data, byte(e.Length))
		} else if e.OpValue == opcodes.OP_PUSHDATA2 {
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(e.Length))
			s.data = append(s.data, b...)
		} else if e.OpValue == opcodes.OP_PUSHDATA4 {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(e.Length))
			s.data = append(s.data, b...)
		}
		if e.OpValue <= opcodes.OP_PUSHDATA4 && e.Length > 0 {
			s.data = append(s.data, e.Data...)
		}
	}
}

func (s *Script) GetData() []byte {
	return s.data
}

func (s *Script) GetBadOpCode() bool {
	return s.badOpCode
}

func (s *Script) convertOPS() error {
	s.ParsedOpCodes = make([]opcodes.ParsedOpCode, 0)
	scriptLen := uint(len(s.data))

	var i uint
	for i < scriptLen {
		var nSize uint
		opcode := s.data[i]
		i++
		if opcode < opcodes.OP_PUSHDATA1 {
			nSize = uint(opcode)
		} else if opcode == opcodes.OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				log.Debug("OP_PUSHDATA1 has no enough data")
				return errors.New("OP_PUSHDATA1 has no enough data")
			}
			nSize = uint(s.data[i])
			i++
		} else if opcode == opcodes.OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				log.Debug("OP_PUSHDATA2 has no enough data")
				return errors.New("OP_PUSHDATA2 has no enough data")
			}
			nSize = uint(binary.LittleEndian.Uint16(s.data[i : i+2]))
			i += 2
		} else if opcode == opcodes.OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				log.Debug("OP_PUSHDATA4 has no enough data")
				return errors.New("OP_PUSHDATA4 has no enough data")

			}
			nSize = uint(binary.LittleEndian.Uint32(s.data[i : i+4]))
			i += 4
		}
		if scriptLen-i < 0 || uint(scriptLen-i) < nSize {
			log.Debug("ConvertOPS script data size is wrong")
			return errors.New("size is wrong")
		}
		parsedopCode := opcodes.NewParsedOpCode(opcode, int(nSize), s.data[i:i+nSize])
		s.ParsedOpCodes = append(s.ParsedOpCodes, *parsedopCode)
		i += nSize
	}
	return nil
}

func (s *Script) RemoveOpcodeByData(data []byte) *Script {
	parsedOpCodes := make([]opcodes.ParsedOpCode, 0, len(s.ParsedOpCodes))
	for _, e := range s.ParsedOpCodes {
		if bytes.Contains(e.Data, data) {
			continue
		}
		parsedOpCodes = append(parsedOpCodes, e)
	}
	return NewScriptOps(parsedOpCodes)
}

func (s *Script) RemoveOpCodeByIndex(index int) *Script {
	opCodesLen := len(s.ParsedOpCodes)
	if index < 0 || index >= opCodesLen {
		return nil
	}
	if index == 0 {
		return NewScriptOps(s.ParsedOpCodes[1 : opCodesLen-1])
	}
	if index == opCodesLen-1 {
		return NewScriptOps(s.ParsedOpCodes[:index])
	}
	parsedOpCodes := make([]opcodes.ParsedOpCode, 0, opCodesLen-1)
	parsedOpCodes = append(parsedOpCodes, s.ParsedOpCodes[:index-1]...)
	parsedOpCodes = append(parsedOpCodes, s.ParsedOpCodes[index+1:opCodesLen-1]...)
	return NewScriptOps(parsedOpCodes)
}

func (s *Script) RemoveOpcode(code byte) *Script {
	parsedOpCodes := make([]opcodes.ParsedOpCode, 0, len(s.ParsedOpCodes))
	for _, e := range s.ParsedOpCodes {
		if e.OpValue == code {
			continue
		}
		parsedOpCodes = append(parsedOpCodes, e)
	}
	return NewScriptOps(parsedOpCodes)
}

func ReadScript(reader io.Reader, maxAllowed uint32, fieldName string) (script []byte, err error) {
	count, err := util.ReadVarInt(reader)
	if err != nil {
		return
	}
	if count > uint64(maxAllowed) {
		log.Debug("ReadScript size err")
		err = errcode.New(errcode.ScriptErrScriptSize)
		return
	}
	//buf := scriptPool.Borrow(count)
	script = make([]byte, count)
	_, err = io.ReadFull(reader, script)
	if err != nil {
		//scriptPool.Return(buf)
		return
	}
	return script, nil

}

func (s *Script) ExtractDestinations(scriptHashAddressID byte) (sType int, addresses []*Address, sigCountRequired int, err error) {
	sType, pubKeys, err := s.CheckScriptPubKeyStandard()
	if err != nil {
		return
	}
	if sType == ScriptPubkey {
		sigCountRequired = 1
		addresses = make([]*Address, 0, 1)
		address, err := AddressFromPublicKey(pubKeys[0])
		if err != nil {
			return sType, nil, 0, err
		}
		addresses = append(addresses, address)
		return sType, addresses, sigCountRequired, nil
	}
	if sType == ScriptPubkeyHash {
		sigCountRequired = 1
		addresses = make([]*Address, 0, 1)
		address, err := AddressFromHash160(pubKeys[0], scriptHashAddressID)
		if err != nil {
			return sType, nil, 0, err
		}
		addresses = append(addresses, address)
		return sType, addresses, sigCountRequired, nil
	}
	if sType == ScriptHash {
		sigCountRequired = 1
		addresses = make([]*Address, 0, 1)
		address, err := AddressFromScriptHash(pubKeys[0])
		if err != nil {
			return sType, nil, 0, err
		}
		addresses = append(addresses, address)
		return sType, addresses, sigCountRequired, nil
	}
	if sType == ScriptMultiSig {
		sigCountRequired = int(pubKeys[0][0])
		addresses = make([]*Address, 0, len(pubKeys)-2)
		for _, e := range pubKeys[1 : len(pubKeys)-2] {
			address, err := AddressFromPublicKey(e)
			if err != nil {
				return sType, nil, 0, err
			}
			addresses = append(addresses, address)
		}
		return
	}
	return
}

func (s *Script) IsCommitment(data []byte) bool {
	if len(data) > 64 || s.Size() != len(data)+2 {
		return false
	}

	if s.data[0] != opcodes.OP_RETURN || int(s.data[1]) != len(data) {
		return false
	}

	for i := 0; i < len(data); i++ {
		if s.data[i+2] != data[i] {
			return false
		}
	}

	return true
}

func BytesToBool(bytes []byte) bool {
	bytesLen := len(bytes)
	if bytesLen == 0 {
		return false
	}
	for i, e := range bytes {
		if uint8(e) != 0 {
			if i == bytesLen-1 && e == 0x80 {
				return false
			}
			return true
		}
	}
	return false
}

func (s *Script) CheckScriptPubKeyStandard() (pubKeyType int, pubKeys [][]byte, err error) {
	//p2sh scriptPubKey
	if s.IsPayToScriptHash() {
		return ScriptHash, nil, nil
	}
	// Provably prunable, data-carrying output
	//
	// So long as script passes the IsUnspendable() test and all but the first
	// byte passes the IsPushOnly() test we don't care what exactly is in the
	// script.
	len := len(s.ParsedOpCodes)
	if len == 0 {
		return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
	}
	parsedOpCode0 := s.ParsedOpCodes[0]
	opValue0 := parsedOpCode0.OpValue

	// OP_RETURN
	if len == 1 {
		if parsedOpCode0.OpValue == opcodes.OP_RETURN {
			return ScriptNullData, nil, nil
		}
		return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
	}

	// OP_RETURN and DATA
	if parsedOpCode0.OpValue == opcodes.OP_RETURN {
		tempScript := NewScriptOps(s.ParsedOpCodes[1:])
		if tempScript.IsPushOnly() {
			return ScriptNullData, nil, nil
		}
		return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
	}

	//PUBKEY OP_CHECKSIG
	if len == 2 {
		if opValue0 > opcodes.OP_PUSHDATA4 || parsedOpCode0.Length < 33 ||
			parsedOpCode0.Length > 65 || s.ParsedOpCodes[1].OpValue != opcodes.OP_CHECKSIG {
			return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
		}
		pubKeyType = ScriptPubkey
		pubKeys = make([][]byte, 0, 1)
		data := parsedOpCode0.Data[:]
		pubKeys = append(pubKeys, data)
		err = nil
		return
	}

	//OP_DUP OP_HASH160 OP_PUBKEYHASH OP_EQUALVERIFY OP_CHECKSIG
	if opValue0 == opcodes.OP_DUP {
		if s.ParsedOpCodes[1].OpValue != opcodes.OP_HASH160 ||
			s.ParsedOpCodes[2].OpValue != opcodes.OP_PUBKEYHASH ||
			s.ParsedOpCodes[2].Length != 20 ||
			s.ParsedOpCodes[3].OpValue != opcodes.OP_EQUALVERIFY ||
			s.ParsedOpCodes[4].OpValue != opcodes.OP_CHECKSIG {
			return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
		}

		pubKeyType = ScriptPubkeyHash
		pubKeys = make([][]byte, 0, 1)
		data := s.ParsedOpCodes[2].Data[:]
		pubKeys = append(pubKeys, data)
		err = nil
		return
	}

	//m pubkey1 pubkey2...pubkeyn n OP_CHECKMULTISIG
	if opValue0 == opcodes.OP_0 || (opValue0 >= opcodes.OP_1 && opValue0 <= opcodes.OP_16) {
		opM := DecodeOPN(opValue0)
		pubKeyCount := 0
		pubKeys = make([][]byte, 0, len-1)
		data := make([]byte, 0, 1)
		data = append(data, byte(opM))
		pubKeys = append(pubKeys, data)
		for i, e := range s.ParsedOpCodes {
			if e.Length >= 33 && e.Length <= 65 {
				pubKeyCount++
				data := s.ParsedOpCodes[i+1].Data[:]
				pubKeys = append(pubKeys, data)
				continue
			}
			opValueI := e.OpValue
			if opValueI == opcodes.OP_0 || (opValue0 >= opcodes.OP_1 && opValue0 <= opcodes.OP_16) {
				opN := DecodeOPN(opValueI)
				// Support up to x-of-3 multisig txns as standard
				if opM < 1 || opN < 1 || opN > 3 || opM > opN || opN != pubKeyCount {
					return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
				}
				data := make([]byte, 0, 1)
				data = append(data, byte(opN))
				pubKeys = append(pubKeys, data)
			} else {
				return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
			}
		}
		if s.ParsedOpCodes[len-1].OpValue != opcodes.OP_CHECKMULTISIG {
			return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
		}
		return ScriptMultiSig, pubKeys, nil
	}

	return ScriptNonStandard, nil, errcode.New(errcode.ScriptErrNonStandard)
}

func (s *Script) CheckScriptSigStandard() error {
	if s.Size() > 1650 {
		log.Debug("ScriptErrSize")
		return errcode.New(errcode.ScriptErrSize)
	}
	if !s.IsPushOnly() {
		//state.Dos(100, false, RejectInvalid, "bad-tx-input-script-not-pushonly", false, "")
		log.Debug("ScriptErrScriptSigNotPushOnly")
		return errcode.New(errcode.ScriptErrScriptSigNotPushOnly)
	}

	return nil
}

func (s *Script) IsPayToScriptHash() bool {
	size := len(s.data)
	return size == 23 &&
		s.data[0] == opcodes.OP_HASH160 &&
		s.data[1] == 0x14 &&
		s.data[22] == opcodes.OP_EQUAL
}

func (s *Script) IsUnspendable() bool {
	return (s.Size() > 0 && s.data[0] == opcodes.OP_RETURN) || s.Size() > MaxScriptSize
}

func (s *Script) IsPushOnly() bool {
	for _, ops := range s.ParsedOpCodes {
		if ops.OpValue > opcodes.OP_16 {
			return false
		}
	}
	return true

}

func (s *Script) GetSigOpCount(accurate bool) int {
	n := 0
	var lastOpcode byte
	for _, e := range s.ParsedOpCodes {
		opcode := e.OpValue
		if opcode == opcodes.OP_CHECKSIG || opcode == opcodes.OP_CHECKSIGVERIFY {
			n++
		} else if opcode == opcodes.OP_CHECKMULTISIG || opcode == opcodes.OP_CHECKMULTISIGVERIFY {
			if accurate && lastOpcode >= opcodes.OP_1 && lastOpcode <= opcodes.OP_16 {
				opn := DecodeOPN(lastOpcode)
				n += opn
			} else {
				n += MaxPubKeysPerMultiSig
			}
		}
		lastOpcode = opcode
	}
	return n
}

func (s *Script) GetP2SHSigOpCount() int {
	// This is a pay-to-script-hash scriptPubKey;
	// get the last item that the scriptSig
	// pushes onto the stack:
	for _, e := range s.ParsedOpCodes {
		opcode := e.OpValue
		if opcode > opcodes.OP_16 {
			return 0
		}
	}
	lastOps := s.ParsedOpCodes[len(s.ParsedOpCodes)-1]
	tempScript := NewScriptRaw(lastOps.Data)
	return tempScript.GetSigOpCount(true)

}

func EncodeOPN(n int) (int, error) {
	if n < 0 || n > 16 {
		return 0, errors.New("EncodeOPN n is out of bounds")
	}
	return opcodes.OP_1 + n - 1, nil
}

func DecodeOPN(opcode byte) int {
	if opcode == opcodes.OP_0 {
		return 0
	}
	if opcode < opcodes.OP_1 || opcode > opcodes.OP_16 {
		panic("Decode Opcode err")
	}
	return int(opcode) - int(opcodes.OP_1-1)
}

func (s *Script) Size() int {
	return len(s.data)
}

func (s *Script) IsEqual(script2 *Script) bool {
	return bytes.Equal(s.data, script2.data)
}

func (s *Script) PushInt64(n int64) error {
	if n == -1 || (n >= 1 && n <= 16) {
		s.data = append(s.data, byte(n+(opcodes.OP_1-1)))
	} else if n == 0 {
		s.data = append(s.data, byte(opcodes.OP_0))
	} else {
		scriptNum := NewScriptNum(n)
		s.data = append(s.data, scriptNum.Serialize()...)
	}
	err := s.convertOPS()
	if err != nil {
		return err
	}
	return nil
}

func (s *Script) PushData(data [][]byte) error {
	for _, e := range data {
		dataLen := len(e)
		if dataLen == 0 {
			s.data = append(s.data, byte(opcodes.OP_0))
		} else if dataLen == 1 && e[0] >= 1 && e[0] <= 16 {
			opN, _ := EncodeOPN(int(e[0]))
			s.data = append(s.data, byte(opN))
		} else if dataLen < opcodes.OP_PUSHDATA1 {
			s.data = append(s.data, byte(dataLen))
		} else if dataLen <= 0xff {
			s.data = append(s.data, opcodes.OP_PUSHDATA1)
			s.data = append(s.data, byte(dataLen))
		} else if dataLen <= 0xffff {
			s.data = append(s.data, opcodes.OP_PUSHDATA2)
			buf := make([]byte, 2)
			binary.LittleEndian.PutUint16(buf, uint16(dataLen))
			s.data = append(s.data, buf...)
		} else {
			s.data = append(s.data, opcodes.OP_PUSHDATA4)
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(dataLen))
			s.data = append(s.data, buf...)
		}
		s.data = append(s.data, e...)
	}
	err := s.convertOPS()
	if err != nil {
		return err
	}

	return nil
}

func CheckSignatureEncoding(vchSig []byte, flags uint32) error {
	// Empty signature. Not strictly DER encoded, but allowed to provide a
	// compact way to provide an invalid signature for use with CHECK(MULTI)SIG
	vchSigLen := len(vchSig)
	if vchSigLen == 0 {
		return nil
	}
	if (flags&(ScriptVerifyDersig|ScriptVerifyLowS|ScriptVerifyStrictEnc)) != 0 &&
		!crypto.IsValidSignatureEncoding(vchSig) {
		log.Debug("ScriptErrInvalidSignatureEncoding")
		return errcode.New(errcode.ScriptErrInvalidSignatureEncoding)

	}
	if (flags & ScriptVerifyLowS) != 0 {
		ret, err := crypto.IsLowDERSignature(vchSig)
		if err != nil || !ret {
			return err
		}
	}

	if (flags & ScriptVerifyStrictEnc) != 0 {
		if !crypto.IsDefineHashtypeSignature(vchSig) {
			log.Debug("ScriptErrSigHashType")
			return errcode.New(errcode.ScriptErrSigHashType)
		}
	}

	return nil
}

func CheckPubKeyEncoding(vchPubKey []byte, flags uint32) error {
	if flags&ScriptVerifyStrictEnc != 0 && !crypto.IsCompressedOrUncompressedPubKey(vchPubKey) {
		log.Debug("ScriptErrPubKeyType")
		return errcode.New(errcode.ScriptErrPubKeyType)

	}
	// Only compressed keys are accepted when
	// ScriptVerifyCompressedPubKeyType is enabled.
	if flags&ScriptVerifyCompressedPubkeyType != 0 && !crypto.IsCompressedPubKey(vchPubKey) {
		log.Debug("ScriptErrNonCompressedPubKey")
		return errcode.New(errcode.ScriptErrNonCompressedPubKey)
	}
	return nil
}

//func (script *Script) GetPubKeyTypeString(t int) string {
//	switch t {
//	case ScriptNonStandard:
//		return "nonstandard"
//	case ScriptPubkey:
//		return "pubkey"
//	case ScriptPubkeyHash:
//		return "pubkeyhash"
//	case ScriptHash:
//		return "scripthash"
//	case ScriptMultiSig:
//		return "multisig"
//	case ScriptNullData:
//		return "nulldata"
//	}
//	return ""
//}

//
//func (script *Script) PushOpCode(opcode int) error {
//	if opcode < 0 || opcode > 0xff {
//		return errors.New("push opcode failed :invalid opcode")
//	}
//	script.data = append(script.data, byte(opcode))
//
//	err := script.convertOPS()
//	if err != nil {
//		return err
//	}
//	return nil
//}

//func (script *Script) PushScriptNum(scriptNum *ScriptNum) {
//	script.data = append(script.data, scriptNum.Serialize()...)
//}

/*
func (script *Script) ParseScript() (stk []ParsedOpCode, err error) {
	stk = make([]ParsedOpCode, 0)
<<<<<<< HEAD
	scriptLen := len(script.data)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.data[i]
=======
	scriptLen := len(script.byteCodes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.byteCodes[i]
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
		parsedopCode := ParsedOpCode{opValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
<<<<<<< HEAD
			parsedopCode.data = script.data[i+1 : i+1+nSize]
=======
			parsedopCode.data = script.byteCodes[i+1 : i+1+nSize]
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7

		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
<<<<<<< HEAD
			nSize = int(script.data[i+1])
			parsedopCode.data = script.data[i+2 : i+2+nSize]
=======
			nSize = int(script.byteCodes[i+1])
			parsedopCode.data = script.byteCodes[i+2 : i+2+nSize]
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
			i++

		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
<<<<<<< HEAD
			nSize = int(binary.LittleEndian.Uint16(script.data[i+1 : i+3]))
			parsedopCode.data = script.data[i+3 : i+3+nSize]
=======
			nSize = int(binary.LittleEndian.Uint16(script.byteCodes[i+1 : i+3]))
			parsedopCode.data = script.byteCodes[i+3 : i+3+nSize]
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
<<<<<<< HEAD
			nSize = int(binary.LittleEndian.Uint32(script.data[i+1 : i+5]))
			parsedopCode.data = script.data[i+5 : i+5+nSize]
=======
			nSize = int(binary.LittleEndian.Uint32(script.byteCodes[i+1 : i+5]))
			parsedopCode.data = script.byteCodes[i+5 : i+5+nSize]
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
			i += 4
		}
		if scriptLen-i < 0 || (scriptLen-i) < nSize {
			err = errors.New("size is wrong")
			return
		}

		stk = append(stk, parsedopCode)
		i += nSize
	}

	return
}
*/
/*
func (script *Script) FindAndDelete(b *Script) (bool, error) {
	//orginalParseCodes, err := script.ParseScript()
	//if err != nil {
	//	return false, err
	//}
	//paramScript, err := b.ParseScript()
	//if err != nil {
	//	return false, err
	//}
	//script.data = make([]byte, 0)

	for i := 0; i < len(orginalParseCodes); i++ {
		isDelete := false
		parseCode := orginalParseCodes[i]
		for j := 0; j < len(paramScript); j++ {
			parseCodeOther := paramScript[j]
			if parseCode.opValue == parseCodeOther.opValue {
				isDelete = true
			}
		}
		if !isDelete {
<<<<<<< HEAD
			script.data = append(script.data, parseCode.opValue)
			script.data = append(script.data, parseCode.data...)
=======
			script.byteCodes = append(script.byteCodes, parseCode.opValue)
			script.byteCodes = append(script.byteCodes, parseCode.data...)
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
		}
	}

	return true, nil
}
*/
/*
func (script *Script) Find(opcode int) bool {
	//stk, err := script.ParseScript()
	//if err != nil {
	//	return false
	//}
	for _, ops := range script.ParsedOpCodes {
		if int(ops.opValue) == opcode {
			return true
		}
	}
	return false
}
*/

/*
func (script *Script) GetSigOpCount() (int, error) {
	if !script.IsPayToScriptHash() {
		return script.GetSigOpCountWithAccurate(true)
	}
	stk, err := script.ParseScript()
	if err != nil {
		return 0, err
	}
	if len(stk) == 0 {
		return 0, nil
	}
	for i := 0; i < len(stk); i++ {
		opcode := stk[i].opValue
		if opcode == OP_16 {
			return 0, nil
		}
	}
	return script.GetSigOpCountWithAccurate(true)
}

func (script *Script) GetSigOpCountFor(scriptSig *Script) (int, error) {
	if !script.IsPayToScriptHash() {
		return script.GetSigOpCountWithAccurate(true)
	}

	// This is a pay-to-script-hash scriptPubKey;
	// get the last item that the scriptSig
	// pushes onto the stack:
	var n = 0
	stk, err := scriptSig.ParseScript()
	if err != nil {
		return n, err
	}

	data := make([]byte, 0)
	for i := 0; i < len(stk); i++ {
		var opcode *byte
		if !scriptSig.GetOp(&i, opcode, &data) {
			return 0, nil
		}

		if *opcode > OP_16 {
			return 0, nil
		}
	}

	subScript := NewScriptRaw(data)
	return subScript.GetSigOpCountWithAccurate(true)
}
*/
/*
func (script *Script) GetScriptByte() []byte {
	scriptByte := make([]byte, 0)
<<<<<<< HEAD
	scriptByte = append(scriptByte, script.data...)
=======
	scriptByte = append(scriptByte, script.byteCodes...)
>>>>>>> c094fa5c6f05ba4ae9dab8c6668ccf09996efbc7
	return scriptByte
}
*/
