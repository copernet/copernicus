package opcodes

import "testing"

func TestGetOpName(t *testing.T) {
	for opCode := 0; opCode < 256; opCode++ {
		opName := GetOpName(opCode)
		switch opCode {
		// push value
		case OP_0:
			if opName != "0" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

		case OP_PUSHDATA1:
			if opName != "OP_PUSHDATA1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_PUSHDATA2:
			if opName != "OP_PUSHDATA2" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_PUSHDATA4:
			if opName != "OP_PUSHDATA4" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_1NEGATE:
			if opName != "-1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_RESERVED:
			if opName != "OP_RESERVED" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_1:
			if opName != "1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2:
			if opName != "2" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_3:
			if opName != "3" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_4:
			if opName != "4" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_5:
			if opName != "5" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_6:
			if opName != "6" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_7:
			if opName != "7" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_8:
			if opName != "8" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_9:
			if opName != "9" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_10:
			if opName != "10" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_11:
			if opName != "11" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_12:
			if opName != "12" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_13:
			if opName != "13" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_14:
			if opName != "14" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_15:
			if opName != "15" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_16:
			if opName != "16" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// control
		case OP_NOP:
			if opName != "OP_NOP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_VER:
			if opName != "OP_VER" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_IF:
			if opName != "OP_IF" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOTIF:
			if opName != "OP_NOTIF" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_VERIF:
			if opName != "OP_VERIF" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_VERNOTIF:
			if opName != "OP_VERNOTIF" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ELSE:
			if opName != "OP_ELSE" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ENDIF:
			if opName != "OP_ENDIF" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_VERIFY:
			if opName != "OP_VERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_RETURN:
			if opName != "OP_RETURN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// stack ops
		case OP_TOALTSTACK:
			if opName != "OP_TOALTSTACK" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_FROMALTSTACK:
			if opName != "OP_FROMALTSTACK" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2DROP:
			if opName != "OP_2DROP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2DUP:
			if opName != "OP_2DUP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_3DUP:
			if opName != "OP_3DUP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2OVER:
			if opName != "OP_2OVER" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2ROT:
			if opName != "OP_2ROT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2SWAP:
			if opName != "OP_2SWAP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_IFDUP:
			if opName != "OP_IFDUP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_DEPTH:
			if opName != "OP_DEPTH" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_DROP:
			if opName != "OP_DROP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_DUP:
			if opName != "OP_DUP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NIP:
			if opName != "OP_NIP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_OVER:
			if opName != "OP_OVER" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_PICK:
			if opName != "OP_PICK" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ROLL:
			if opName != "OP_ROLL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ROT:
			if opName != "OP_ROT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SWAP:
			if opName != "OP_SWAP" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_TUCK:
			if opName != "OP_TUCK" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// splice ops
		case OP_CAT:
			if opName != "OP_CAT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SPLIT:
			if opName != "OP_SPLIT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NUM2BIN:
			if opName != "OP_NUM2BIN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_BIN2NUM:
			if opName != "OP_BIN2NUM" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SIZE:
			if opName != "OP_SIZE" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// bit logic
		case OP_INVERT:
			if opName != "OP_INVERT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_AND:
			if opName != "OP_AND" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_OR:
			if opName != "OP_OR" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_XOR:
			if opName != "OP_XOR" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_EQUAL:
			if opName != "OP_EQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_EQUALVERIFY:
			if opName != "OP_EQUALVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_RESERVED1:
			if opName != "OP_RESERVED1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_RESERVED2:
			if opName != "OP_RESERVED2" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// numeric
		case OP_1ADD:
			if opName != "OP_1ADD" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_1SUB:
			if opName != "OP_1SUB" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2MUL:
			if opName != "OP_2MUL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_2DIV:
			if opName != "OP_2DIV" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NEGATE:
			if opName != "OP_NEGATE" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ABS:
			if opName != "OP_ABS" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOT:
			if opName != "OP_NOT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_0NOTEQUAL:
			if opName != "OP_0NOTEQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_ADD:
			if opName != "OP_ADD" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SUB:
			if opName != "OP_SUB" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_MUL:
			if opName != "OP_MUL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_DIV:
			if opName != "OP_DIV" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_MOD:
			if opName != "OP_MOD" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_LSHIFT:
			if opName != "OP_LSHIFT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_RSHIFT:
			if opName != "OP_RSHIFT" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_BOOLAND:
			if opName != "OP_BOOLAND" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_BOOLOR:
			if opName != "OP_BOOLOR" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NUMEQUAL:
			if opName != "OP_NUMEQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NUMEQUALVERIFY:
			if opName != "OP_NUMEQUALVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NUMNOTEQUAL:
			if opName != "OP_NUMNOTEQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_LESSTHAN:
			if opName != "OP_LESSTHAN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_GREATERTHAN:
			if opName != "OP_GREATERTHAN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_LESSTHANOREQUAL:
			if opName != "OP_LESSTHANOREQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_GREATERTHANOREQUAL:
			if opName != "OP_GREATERTHANOREQUAL" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_MIN:
			if opName != "OP_MIN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_MAX:
			if opName != "OP_MAX" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_WITHIN:
			if opName != "OP_WITHIN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// crypto
		case OP_RIPEMD160:
			if opName != "OP_RIPEMD160" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SHA1:
			if opName != "OP_SHA1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_SHA256:
			if opName != "OP_SHA256" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_HASH160:
			if opName != "OP_HASH160" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_HASH256:
			if opName != "OP_HASH256" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CODESEPARATOR:
			if opName != "OP_CODESEPARATOR" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKSIG:
			if opName != "OP_CHECKSIG" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKSIGVERIFY:
			if opName != "OP_CHECKSIGVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKMULTISIG:
			if opName != "OP_CHECKMULTISIG" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKMULTISIGVERIFY:
			if opName != "OP_CHECKMULTISIGVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// expansion
		case OP_NOP1:
			if opName != "OP_NOP1" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKLOCKTIMEVERIFY:
			if opName != "OP_CHECKLOCKTIMEVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_CHECKSEQUENCEVERIFY:
			if opName != "OP_CHECKSEQUENCEVERIFY" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP4:
			if opName != "OP_NOP4" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP5:
			if opName != "OP_NOP5" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP6:
			if opName != "OP_NOP6" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP7:
			if opName != "OP_NOP7" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP8:
			if opName != "OP_NOP8" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP9:
			if opName != "OP_NOP9" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		case OP_NOP10:
			if opName != "OP_NOP10" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

		case OP_INVALIDOPCODE:
			if opName != "OP_INVALIDOPCODE" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}

			// Note:
			//  The template matching params OP_SMALLINTEGER/etc are defined in opcodetype enum
			//  as kind of implementation hack, they are *NOT* real opcodes.  If found in real
			//  Script, just let the default: case deal with them.

		default:
			if opName != "OP_UNKNOWN" {
				t.Errorf("GetOpName return error opName of opCode: %d", opCode)
			}
		}
	}
}
