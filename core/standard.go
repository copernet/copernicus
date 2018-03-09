package core

import (
	"bytes"

	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/utils"
)

const (
	TX_NONSTANDARD = iota

	TX_PUBKEY
	TX_PUBKEYHASH
	TX_SCRIPTHASH
	TX_MULTISIG
	TX_NULL_DATA

	DEFAULT_ACCEPT_DATACARRIER bool = true
	MAX_OP_RETURN_RELAY        uint = 83
)

/*Solver Return public keys or hashes from scriptPubKey, for 'standard' transaction
 * types.
 */
func Solver(scriptPubKey *Script, typeRet *int, vSolutionsRet *container.Vector) bool {
	// Templates
	mTemplates := make(map[int]Script)
	if len(mTemplates) == 0 {
		// Standard tx, sender provides pubkey, receiver adds signature[]byte{byte(model.OP_PUBKEY), byte(model.OP_CHECKSIG)
		scriptBytes := []byte{byte(OP_PUBKEY), byte(OP_CHECKSIG)}
		mTemplates[TX_PUBKEY] = *NewScriptRaw(scriptBytes)

		// Bitcoin address tx, sender provides hash of pubkey, receiver provides
		// signature and pubkey
		scriptBytes = []byte{byte(OP_DUP), byte(OP_HASH160), byte(OP_PUBKEYHASH),
			byte(OP_EQUALVERIFY), byte(OP_CHECKSIG)}
		mTemplates[TX_PUBKEYHASH] = *NewScriptRaw(scriptBytes)

		// Sender provides N pubkeys, receivers provides M signatures
		scriptBytes = []byte{byte(OP_SMALLINTEGER), byte(OP_PUBKEYS),
			byte(OP_SMALLINTEGER), byte(OP_CHECKMULTISIG)}
		mTemplates[TX_MULTISIG] = *NewScriptRaw(scriptBytes)
	}

	vSolutionsRet.Clear()

	// Shortcut for pay-to-script-hash, which are more constrained than the
	// other types:
	// it is always OP_HASH160 20 [20 byte hash] OP_EQUAL
	scriptByte := scriptPubKey.GetScriptByte()
	if scriptPubKey.IsPayToScriptHash() {
		*typeRet = TX_SCRIPTHASH
		hashBytes := container.NewVector()
		hashBytes.PushBack(scriptByte[2:22])
		vSolutionsRet.PushBack(hashBytes)
		return true
	}

	// Provably prunable, data-carrying output
	//
	// So long as script passes the IsUnspendable() test and all but the first
	// byte passes the IsPushOnly() test we don't care what exactly is in the
	// script.
	tmpScriptPubkey := NewScriptRaw(scriptByte[1:])
	if scriptPubKey.Size() >= 1 && scriptByte[0] == OP_RETURN && tmpScriptPubkey.IsPushOnly() {
		*typeRet = TX_NULL_DATA
		return true
	}

	// Scan templates
	script1 := scriptPubKey
	for _, tmpScript := range mTemplates {

		vSolutionsRet.Clear()

		var (
			vch1    []byte
			vch2    []byte
			opcode1 byte
			opcode2 byte
			pc1     int
			pc2     int
		)

		for {
			if pc1 == script1.Size() && pc2 == tmpScript.Size() {
				// Found a match
				*typeRet = int(tmpScript.GetScriptByte()[0])
				if *typeRet == TX_MULTISIG {
					// Additional checks for TX_MULTISIG:
					front := vSolutionsRet.Array[0].([]byte)
					m := front[0]
					back := vSolutionsRet.Array[vSolutionsRet.Size()-1].([]byte)
					n := back[0]
					if m < 1 || n < 1 || m > n || vSolutionsRet.Size()-2 != int(n) {
						return false
					}
				}
				return true
			}

			if !script1.GetOp(&pc1, &opcode1, &vch1) {
				break
			}
			if !tmpScript.GetOp(&pc2, &opcode2, &vch2) {
				break
			}

			// Template matching opcodes:
			if opcode2 == OP_PUBKEYS {
				for len(vch1) >= 33 && len(vch1) <= 65 {
					vSolutionsRet.PushBack(vch1)
					if !script1.GetOp(&pc1, &opcode1, &vch1) {
						break
					}
				}
				if !tmpScript.GetOp(&pc2, &opcode2, &vch2) {
					break
				}
				// Normal situation is to fall through to other if/else
				// statements
			}

			if opcode2 == OP_PUBKEY {
				if len(vch1) < 33 || len(vch1) > 65 {
					break
				}
				vSolutionsRet.PushBack(vch1)
			} else if opcode2 == OP_PUBKEYHASH {
				if len(vch1) != utils.Hash160Size {
					break
				}
				vSolutionsRet.PushBack(vch1)
			} else if opcode2 == OP_SMALLINTEGER {
				// Single-byte small integer pushed onto vSolutions
				if opcode1 == OP_0 || opcode1 >= OP_1 && opcode1 <= OP_16 {
					n, err := DecodeOPN(int(opcode1))
					if err != nil {
						return false
					}
					valType := []byte{1, byte(n)}
					vSolutionsRet.PushBack(valType)
				} else {
					break
				}
			} else if opcode1 != opcode2 || !bytes.Equal(vch1, vch2) {
				// Others must match exactly
				break
			}

		}

	}
	vSolutionsRet.Clear()
	*typeRet = TX_NONSTANDARD
	return false

}
