package script

type TxScriptFlags uint32

const (
	TxScriptBip16 TxScriptFlags = 1 << iota
	TxScriptStrictMultiSig
	TxScriptDiscourageUpgradeableNops
	TxScriptVerifyCheckLockTimeVerify
	// TxScriptVerifyCheckSequenceVerify defines whether to allow execution
	// pathways of a script to be restricted based on the age of the output
	// being spent.  This is BIP0112.
	TxScriptVerifyCheckSequenceVerify

	// TxScriptVerifyCleanStack defines that the stack must contain only
	// one stack element after evaluation and that the element must be
	// true if interpreted as a boolean.  This is rule 6 of BIP0062.
	// This flag should never be used without the TxScriptBip16 flag.
	TxScriptVerifyCleanStack

	// TxScriptVerifyDERSignatures defines that signatures are required
	// to compily with the DER format.
	TxScriptVerifyDERSignatures

	// TxScriptVerifyLowS defines that signtures are required to comply with
	// the DER format and whose S value is <= order / 2.  This is rule 5
	// of BIP0062.
	TxScriptVerifyLowS
	// TxScriptVerifyMinimalData defines that signatures must use the smallest
	// push operator. This is both rules 3 and 4 of BIP0062.
	TxScriptVerifyMinimalData

	// TxScriptVerifyNullFail defines that signatures must be empty if
	// a CHECKSIG or CHECKMULTISIG operation fails.
	TxScriptVerifyNullFail

	// TxScriptVerifySigPushOnly defines that signature scripts must contain
	// only pushed data.  This is rule 2 of BIP0062.
	TxScriptVerifySigPushOnly

	// TxScriptVerifyStrictEncoding defines that signature scripts and
	// public keys must follow the strict encoding requirements.
	TxScriptVerifyStrictEncoding
)
