// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

// Standard JSON-RPC 2.0 errors.
var (
	ErrRPCInvalidRequest = &RPCError{
		Code:    -32600,
		Message: "Invalid request",
	}
	ErrRPCMethodNotFound = &RPCError{
		Code:    -32601,
		Message: "Method not found",
	}
	ErrRPCInvalidParams = &RPCError{
		Code:    -32602,
		Message: "Invalid parameters",
	}
	ErrRPCInternal = &RPCError{
		Code:    -32603,
		Message: "Internal error",
	}
	ErrRPCParse = &RPCError{
		Code:    -32700,
		Message: "Parse error",
	}
)

// General application defined JSON errors.
const (
	ErrRPCMisc                RPCErrorCode = -1
	ErrRPCForbiddenBySafeMode RPCErrorCode = -2
	ErrRPCType                RPCErrorCode = -3
	ErrRPCInvalidAddressOrKey RPCErrorCode = -5
	ErrRPCOutOfMemory         RPCErrorCode = -7
	ErrRPCInvalidParameter    RPCErrorCode = -8
	ErrRPCDatabase            RPCErrorCode = -20
	ErrRPCDeserialization     RPCErrorCode = -22
	ErrRPCVerify              RPCErrorCode = -25
)

// Peer-to-peer client errors.
const (
	ErrRPCClientNotConnected      RPCErrorCode = -9
	ErrRPCClientInInitialDownload RPCErrorCode = -10
	ErrRPCClientNodeNotAdded      RPCErrorCode = -24
)

// Wallet JSON errors
const (
	ErrRPCWallet                    RPCErrorCode = -4
	ErrRPCWalletInsufficientFunds   RPCErrorCode = -6
	ErrRPCWalletInvalidAccountName  RPCErrorCode = -11
	ErrRPCWalletKeypoolRanOut       RPCErrorCode = -12
	ErrRPCWalletUnlockNeeded        RPCErrorCode = -13
	ErrRPCWalletPassphraseIncorrect RPCErrorCode = -14
	ErrRPCWalletWrongEncState       RPCErrorCode = -15
	ErrRPCWalletEncryptionFailed    RPCErrorCode = -16
	ErrRPCWalletAlreadyUnlocked     RPCErrorCode = -17
)

// Specific Errors related to commands.  These are the ones a user of the RPC
// server are most likely to see.  Generally, the codes should match one of the
// more general errors above.
const (
	ErrRPCBlockNotFound     RPCErrorCode = -5
	ErrRPCBlockCount        RPCErrorCode = -5
	ErrRPCBestBlockHash     RPCErrorCode = -5
	ErrRPCDifficulty        RPCErrorCode = -5
	ErrRPCOutOfRange        RPCErrorCode = -1
	ErrRPCNoTxInfo          RPCErrorCode = -5
	ErrRPCNoNewestBlockInfo RPCErrorCode = -5
	ErrRPCInvalidTxVout     RPCErrorCode = -5
	ErrRPCDecodeHexString   RPCErrorCode = -8
)

// Errors that are specific to btcd.
const (
	ErrRPCNoWallet      RPCErrorCode = -1
	ErrRPCUnimplemented RPCErrorCode = -1
)

const (
	ErrUnDefined        RPCErrorCode = 404 // custom rpc error code
	ErrInvalidParameter RPCErrorCode = -30

	// BCH v0.16.0
	// Standard JSON-RPC 2.0 errors

	// RPCInvalidRequest is internally mapped to HTTP_BAD_REQUEST (400).
	// It should not be used for application-layer errors.
	RPCInvalidRequest = -32600
	// RPCMethodNotFound is internally mapped to HTTP_NOT_FOUND (404).
	// It should not be used for application-layer errors.
	RPCMethodNotFound = -32601
	RPCInvalidParams  = -32602
	// RPCInternalError should only be used for genuine errors in bitcoind
	// (for exampled datadir corruption).
	RPCInternalError = -32603
	RPCParseError    = -32700

	// RPCMiscError General application defined errors
	// std::exception thrown in command handling
	RPCMiscError = -1
	// RPCForbiddenBySafeMode Server is in safe mode, and command is not allowed in safe mode
	RPCForbiddenBySafeMode = -2
	// RPCTypeError Unexpected type was passed as parameter
	RPCTypeError = -3
	// RPCInvalidAddressOrKey Invalid address or key
	RPCInvalidAddressOrKey = -5
	// RPCOutOfMemory Ran out of memory during operation
	RPCOutOfMemory = -7
	// RPCInvalidParameter Invalid, missing or duplicate parameter
	RPCInvalidParameter = -8
	// RPCDatabaseError Database error
	RPCDatabaseError = -20
	// RPCDeserializationError Error parsing or validating structure in raw format
	RPCDeserializationError = -22
	// RPCVerifyError General error during transaction or block submission
	RPCVerifyError = -25
	// RPCVerifyRejected Transaction or block was rejected by network rules
	RPCVerifyRejected = -26
	// RPCVerifyAlreadyInChain Transaction already in chain
	RPCVerifyAlreadyInChain = -27
	// RPCInWarmup Client still warming up
	RPCInWarmup = -28

	// RPCTransactionError ... Aliases for backward compatibility
	RPCTransactionError          = RPCVerifyError
	RPCTransactionRejected       = RPCVerifyRejected
	RPCTransactionAlreadyInChain = RPCVerifyAlreadyInChain

	// RPCClientNotConnected ... P2P client errors
	// Bitcoin is not connected
	RPCClientNotConnected = -9
	// RPCClientInInitialDownload Still downloading initial blocks
	RPCClientInInitialDownload = -10
	// RPCClientNodeAlreadyAdded Node is already added
	RPCClientNodeAlreadyAdded = -23
	// RPCClientNodeNotAdded Node has not been added before
	RPCClientNodeNotAdded = -24
	// RPCClientNodeNotConnected Node to disconnect not found in connected nodes
	RPCClientNodeNotConnected = -29
	// RPCClientInvalidIPOrSubnet Invalid IP/Subnet
	RPCClientInvalidIPOrSubnet = -30
	// RPCClientP2pDisabled No valid connection manager instance found
	RPCClientP2pDisabled = -31

	// RPCWalletError ... Wallet errors
	// Unspecified problem with wallet (key not found etc.)
	RPCWalletError = -4
	// RPCWalletInsufficientFunds Not enough funds in wallet or account
	RPCWalletInsufficientFunds = -6
	// RPCWalletInvalidAccountName Invalid account name
	RPCWalletInvalidAccountName = -11
	// RPCWalletKeypoolRanOut Keypool ran out, call keypoolrefill first
	RPCWalletKeypoolRanOut = -12
	// RPCWalletUnlockNeeded Enter the wallet passphrase with walletpassphrase first
	RPCWalletUnlockNeeded = -13
	// RPCWalletPassphraseIncorrect The wallet passphrase entered was incorrect
	RPCWalletPassphraseIncorrect = -14
	// RPCWalletWrongEncState = -15 Command given in wrong wallet encryption state (encrypting an encrypted wallet etc.)
	RPCWalletWrongEncState = -15
	// RPCWalletEncryptionFailed Failed to encrypt the wallet
	RPCWalletEncryptionFailed = -16
	// RPCWalletAlreadyUnlocked Wallet is already unlocked
	RPCWalletAlreadyUnlocked = -17
)
