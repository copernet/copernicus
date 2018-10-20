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

	// MaxStandardScriptSigSize is
	// Biggest 'standard' txin is a 15-of-15 P2SH multisig with compressed
	// keys (remember the 520 byte limit on redeemScript size). That works
	// out to a (15*(33+1))+3=513 byte redeemScript, 513+1+15*(73+1)+3=1627
	// bytes of scriptSig, which we round off to 1650 bytes for some minor
	// future-proofing. That's also enough to spend a 20-of-20 CHECKMULTISIG
	// scriptPubKey, though such a scriptPubKey is not considered standard.
	MaxStandardScriptSigSize = 1650
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
	// Is OP_CHECKDATASIG and variant are enabled.
	//
	//ScriptEnableCheckDataSig = (1 << 18)

	ScriptMaxOpReturnRelay uint = 223
)

const (
	ScriptNonStandard = iota
	// ScriptPubkey and following are 'standard' transaction types:
	ScriptPubkey
	ScriptPubkeyHash
	ScriptHash
	ScriptMultiSig
	ScriptNullData
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
	return err
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
	s.convertOPS()
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

func (s *Script) convertOPS() (err error) {
	s.ParsedOpCodes = make([]opcodes.ParsedOpCode, 0)
	scriptLen := uint(len(s.data))
	err = nil

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
				err = errors.New("OP_PUSHDATA1 has no enough data")
				break
			}
			nSize = uint(s.data[i])
			i++
		} else if opcode == opcodes.OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				log.Debug("OP_PUSHDATA2 has no enough data")
				err = errors.New("OP_PUSHDATA2 has no enough data")
				break
			}
			nSize = uint(binary.LittleEndian.Uint16(s.data[i : i+2]))
			i += 2
		} else if opcode == opcodes.OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				log.Debug("OP_PUSHDATA4 has no enough data")
				err = errors.New("OP_PUSHDATA4 has no enough data")
				break

			}
			nSize = uint(binary.LittleEndian.Uint32(s.data[i : i+4]))
			i += 4
		}
		if scriptLen-i < 0 || scriptLen-i < nSize {
			log.Debug("ConvertOPS script data size is wrong")
			err = errors.New("size is wrong")
			break
		}
		parsedopCode := opcodes.NewParsedOpCode(opcode, int(nSize), s.data[i:i+nSize])
		s.ParsedOpCodes = append(s.ParsedOpCodes, *parsedopCode)
		i += nSize
	}
	if err != nil {
		s.badOpCode = true
	} else {
		s.badOpCode = false
	}
	return
}

func (s *Script) RemoveOpcodeByData(data []byte) *Script {
	parsedOpCodes := make([]opcodes.ParsedOpCode, 0, len(s.ParsedOpCodes))
	for _, e := range s.ParsedOpCodes {
		if e.CheckCompactDataPush() && bytes.Equal(e.Data, data) {
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
		if opCodesLen == 1 {
			return NewEmptyScript()
		}
		return NewScriptOps(s.ParsedOpCodes[1:])
	}
	if index == opCodesLen-1 {
		return NewScriptOps(s.ParsedOpCodes[:index])
	}
	parsedOpCodes := make([]opcodes.ParsedOpCode, 0, opCodesLen-1)
	parsedOpCodes = append(parsedOpCodes, s.ParsedOpCodes[:index-1]...)
	parsedOpCodes = append(parsedOpCodes, s.ParsedOpCodes[index+1:]...)
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

func (s *Script) ExtractDestinations() (sType int, addresses []*Address, sigCountRequired int, err error) {
	sType, pubKeys, isStandard := s.IsStandardScriptPubKey()
	if !isStandard {
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
		address, err := AddressFromHash160(pubKeys[0], AddressVerPubKey())
		if err != nil {
			return sType, nil, 0, err
		}
		addresses = append(addresses, address)
		return sType, addresses, sigCountRequired, nil
	}
	if sType == ScriptHash {
		sigCountRequired = 1
		addresses = make([]*Address, 0, 1)
		address, err := AddressFromHash160(pubKeys[0], AddressVerScript())
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
		if e != 0 {
			if i == bytesLen-1 && e == 0x80 {
				return false
			}
			return true
		}
	}
	return false
}

func (s *Script) IsStandardScriptPubKey() (pubKeyType int, pubKeys [][]byte, isStandard bool) {
	//p2sh scriptPubKey
	if s.IsPayToScriptHash() {
		return ScriptHash, [][]byte{s.ParsedOpCodes[1].Data}, true
	}
	// Provably prunable, data-carrying output
	//
	// So long as script passes the IsUnspendable() test and all but the first
	// byte passes the IsPushOnly() test we don't care what exactly is in the
	// script.
	opCodesLen := len(s.ParsedOpCodes)
	if opCodesLen == 0 {
		return ScriptNonStandard, nil, false
	}
	parsedOpCode0 := s.ParsedOpCodes[0]
	opValue0 := parsedOpCode0.OpValue

	// OP_RETURN
	if opCodesLen == 1 {
		if parsedOpCode0.OpValue == opcodes.OP_RETURN {
			return ScriptNullData, nil, true
		}
		return ScriptNonStandard, nil, false
	}

	// OP_RETURN and DATA
	if parsedOpCode0.OpValue == opcodes.OP_RETURN {
		tempScript := NewScriptOps(s.ParsedOpCodes[1:])
		if tempScript.IsPushOnly() {
			return ScriptNullData, nil, true
		}
		return ScriptNonStandard, nil, false
	}

	//PUBKEY OP_CHECKSIG
	if opCodesLen == 2 {
		if opValue0 > opcodes.OP_PUSHDATA4 || parsedOpCode0.Length < 33 ||
			parsedOpCode0.Length > 65 || s.ParsedOpCodes[1].OpValue != opcodes.OP_CHECKSIG {
			return ScriptNonStandard, nil, false
		}
		pubKeyType = ScriptPubkey
		pubKeys = make([][]byte, 0, 1)
		data := parsedOpCode0.Data
		pubKeys = append(pubKeys, data)
		isStandard = true
		return
	}

	//OP_DUP OP_HASH160 PUBKEYHASH OP_EQUALVERIFY OP_CHECKSIG
	if opValue0 == opcodes.OP_DUP {
		if opCodesLen != 5 {
			return ScriptNonStandard, nil, false
		}
		if s.ParsedOpCodes[1].OpValue != opcodes.OP_HASH160 ||
			s.ParsedOpCodes[2].Length != 20 ||
			s.ParsedOpCodes[3].OpValue != opcodes.OP_EQUALVERIFY ||
			s.ParsedOpCodes[4].OpValue != opcodes.OP_CHECKSIG {
			return ScriptNonStandard, nil, false
		}

		pubKeyType = ScriptPubkeyHash
		pubKeys = make([][]byte, 0, 1)
		data := s.ParsedOpCodes[2].Data
		pubKeys = append(pubKeys, data)
		isStandard = true
		return
	}

	//m pubkey1 pubkey2...pubkeyn n OP_CHECKMULTISIG
	if opValue0 == opcodes.OP_0 || (opValue0 >= opcodes.OP_1 && opValue0 <= opcodes.OP_16) {
		if opCodesLen < 4 {
			return ScriptNonStandard, nil, false
		}
		opM := DecodeOPN(opValue0)
		pubKeys = make([][]byte, 0, opCodesLen-1)
		data := make([]byte, 0, 1)
		data = append(data, byte(opM))
		pubKeys = append(pubKeys, data)
		for _, e := range s.ParsedOpCodes[1 : opCodesLen-2] {
			if e.Length >= 33 && e.Length <= 65 {
				//data := s.ParsedOpCodes[i+1].Data[:]
				//data := s.ParsedOpCodes[i].Data[:]
				pubKeys = append(pubKeys, e.Data)
				continue
			}
			return ScriptNonStandard, nil, false
		}

		opN := 0
		opValueI := s.ParsedOpCodes[opCodesLen-2].OpValue
		if opValueI == opcodes.OP_0 || (opValueI >= opcodes.OP_1 && opValueI <= opcodes.OP_16) {
			opN = DecodeOPN(opValueI)
			data := make([]byte, 0, 1)
			data = append(data, byte(opN))
			pubKeys = append(pubKeys, data)
		} else {
			return ScriptNonStandard, nil, false
		}
		if s.ParsedOpCodes[opCodesLen-1].OpValue != opcodes.OP_CHECKMULTISIG {
			return ScriptNonStandard, nil, false
		}
		// Support up to x-of-3 multisig txns as standard
		if opM < 1 || opN < 1 || opM > opN || opN != len(pubKeys)-2 {
			return ScriptMultiSig, nil, false
		}
		return ScriptMultiSig, pubKeys, true
	}

	return ScriptNonStandard, nil, false
}

func (s *Script) CheckScriptSigStandard() (bool, string) {
	if s.Size() > MaxStandardScriptSigSize {
		return false, "scriptsig-size"
	}
	if !s.IsPushOnly() {
		return false, "scriptsig-not-pushonly"
	}

	return true, ""
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
	if s.badOpCode {
		return false
	}
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
			//} else if opcode == opcodes.OP_CHECKDATASIG || opcode == opcodes.OP_CHECKDATASIGVERIFY {
			//	if flags&ScriptEnableCheckDataSig == ScriptEnableCheckDataSig {
			//		n++
			//	}
		} else if opcode == opcodes.OP_CHECKMULTISIG || opcode == opcodes.OP_CHECKMULTISIGVERIFY {
			if accurate && lastOpcode >= opcodes.OP_1 && lastOpcode <= opcodes.OP_16 {
				n += DecodeOPN(lastOpcode)
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
	if s.badOpCode {
		return 0
	}
	for _, e := range s.ParsedOpCodes {
		opcode := e.OpValue
		if opcode > opcodes.OP_16 {
			return 0
		}
	}
	lastOps := s.ParsedOpCodes[len(s.ParsedOpCodes)-1]
	tempScript := NewScriptRaw(lastOps.Data)
	//return tempScript.GetSigOpCount(flags, true)
	return tempScript.GetSigOpCount(true)

}

func EncodeOPN(n int) (int, error) {
	if n < 0 || n > 16 {
		return 0, errors.New("EncodeOPN n is out of bounds")
	}

	if n == 0 {
		return opcodes.OP_0, nil
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

/*
func (s *Script) FindAndDelete(b *Script) int {
	var (
		nFound int
		pc, pcPre uint64
		result Script
	)
	if len(b.data) == 0 {
		return nFound
	}
	for {
		for pc + uint64(len(b.data)) <= uint64(len(s.data)) && bytes.Equal(b.data, s.data[pc: pc + uint64(len(b.data))]) {
			nFound++
			pc = pc + uint64(len(b.data))
		}
		pcPre = pc
		if !s.getOp(&pc) {
			break
		}
		result.data = bytes.Join([][]byte{result.data, s.data[pcPre: pc]}, []byte(""))
	}
	result.data = bytes.Join([][]byte{result.data, s.data[pcPre:]}, []byte(""))
	*s = result
	s.convertOPS()
	return nFound
}

func (s *Script) getOp(pc *uint64) bool {
	if *pc >= uint64(len(s.data)) {
		return false
	}
	opcode := uint64(s.data[*pc])
	*pc++
	if opcode < opcodes.OP_PUSHDATA1 {
		*pc += opcode
	} else if opcode == opcodes.OP_PUSHDATA1 {
		if *pc >= uint64(len(s.data)) {
			return false
		}
		*pc += opcode + 1 + uint64(s.data[*pc])
	} else if opcode == opcodes.OP_PUSHDATA2 {
		if *pc >= uint64(len(s.data)) - 1 {
			return false
		}
		*pc += opcode + 2 + binary.LittleEndian.Uint64(s.data[*pc: 2 + *pc])
	} else if opcode == opcodes.OP_PUSHDATA4 {
		if *pc >= uint64(len(s.data)) - 3 {
			return false
		}
		*pc += opcode + 4 + binary.LittleEndian.Uint64(s.data[*pc: 4 + *pc])
	}
	return *pc <= uint64(len(s.data))
}
*/

func (s *Script) PushOpCode(n int) error {
	if n < 0 || n > 0xff {
		return errcode.New(errcode.ScriptErrInvalidOpCode)
	}
	s.data = append(s.data, byte(n))
	err := s.convertOPS()
	return err
}

func (s *Script) PushInt64(n int64) error {
	if n >= -1 && n <= 16 {
		if n == -1 || (n >= 1 && n <= 16) {
			s.data = append(s.data, byte(n+(opcodes.OP_1-1)))
		} else if n == 0 {
			s.data = append(s.data, byte(opcodes.OP_0))
		}
		err := s.convertOPS()
		return err
	}

	scriptNum := NewScriptNum(n)
	err := s.PushScriptNum(scriptNum)

	return err
}

func (s *Script) PushScriptNum(sn *ScriptNum) error {
	err := s.PushSingleData(sn.Serialize())
	return err
}

func (s *Script) PushData(data []byte) error {
	s.data = append(s.data, data...)
	return s.convertOPS()
}

func (s *Script) PushSingleData(data []byte) error {
	dataLen := len(data)
	if dataLen < opcodes.OP_PUSHDATA1 {
		s.data = append(s.data, byte(dataLen))
	} else if dataLen <= 0xff {
		s.data = append(s.data, opcodes.OP_PUSHDATA1, byte(dataLen))
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
	s.data = append(s.data, data...)
	err := s.convertOPS()
	return err
}

func (s *Script) PushMultData(data [][]byte) error {
	for _, e := range data {
		dataLen := len(e)
		if dataLen < opcodes.OP_PUSHDATA1 {
			s.data = append(s.data, byte(dataLen))
		} else if dataLen <= 0xff {
			s.data = append(s.data, opcodes.OP_PUSHDATA1, byte(dataLen))
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
	return err
}

func (s *Script) Bytes() []byte {
	return s.data
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
		return errcode.New(errcode.ScriptErrSigDer)

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
		hashType := vchSig[len(vchSig)-1]
		useForkID := false
		forkIDEnable := false
		if hashType&crypto.SigHashForkID != 0 {
			useForkID = true
		}
		if flags&ScriptEnableSigHashForkID != 0 {
			forkIDEnable = true
		}
		if !forkIDEnable && useForkID {
			return errcode.New(errcode.ScriptErrIllegalForkID)
		}
		if forkIDEnable && !useForkID {
			return errcode.New(errcode.ScriptErrMustUseForkID)
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

func IsOpCodeDisabled(opCode byte, flags uint32) bool {
	switch opCode {
	case opcodes.OP_INVERT:
		fallthrough
	case opcodes.OP_2MUL:
		fallthrough
	case opcodes.OP_2DIV:
		fallthrough
	case opcodes.OP_MUL:
		fallthrough
	case opcodes.OP_LSHIFT:
		fallthrough
	case opcodes.OP_RSHIFT:
		return true

	case opcodes.OP_CAT:
		fallthrough
	case opcodes.OP_SPLIT:
		fallthrough
	case opcodes.OP_AND:
		fallthrough
	case opcodes.OP_OR:
		fallthrough
	case opcodes.OP_XOR:
		fallthrough
	case opcodes.OP_NUM2BIN:
		fallthrough
	case opcodes.OP_BIN2NUM:
		fallthrough
	case opcodes.OP_DIV:
		fallthrough
	case opcodes.OP_MOD:
		// Opcodes that have been reenabled.
		if (flags & ScriptEnableMonolithOpcodes) == 0 {
			return true
		}
	default:
	}
	return false
}
