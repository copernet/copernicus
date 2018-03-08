package model

const (
	// BLOCK_VALID_UNKNOWN : Unused.
	BLOCK_VALID_UNKNOWN uint32 = 0

	// BLOCK_VALID_HEADER : Parsed , version ok , hash satisfies claimed PoW, 1 <= vtx count <=max
	// timestamp not in future
	BLOCK_VALID_HEADER uint32 = 1

	// BLOCK_VALID_TREE : All parent headers found, difficulty matches, timestamp>= median
	// previous , checkpoint , Implies all parents are also at least TREE
	BLOCK_VALID_TREE uint32 = 2

	// BLOCK_VALID_TRANSACTIONS : Only first tx is coinbase, 2 <= coinbase input script length <= 100,
	// transactions valid, no duplicate txids , sigops , size , merkle root .
	// Implies all parents are at least TREE but not necessarily TRANSACTIONS.
	// When all parent blocks also have TRANSACTIONS , CBlockIndex ::nChainTx wll be set
	BLOCK_VALID_TRANSACTIONS uint32 = 3

	// BLOCK_VALID_CHAIN : Outputs do not overspend inputs , no double spends , coinbase output ok
	// no immature coinbase spends , BIP30.
	// Implies all parents are also at least CHAIN.
	BLOCK_VALID_CHAIN uint32 = 4

	// BLOCK_VALID_SCRIPTS : Scripts & Signatures ok. Implies all parents are also at least SCRIPTS.
	BLOCK_VALID_SCRIPTS uint32 = 5

	// BLOCK_VALID_MASK : All validity bits
	BLOCK_VALID_MASK uint32 = BLOCK_VALID_HEADER |
		BLOCK_VALID_TREE |
		BLOCK_VALID_TRANSACTIONS |
		BLOCK_VALID_CHAIN |
		BLOCK_VALID_SCRIPTS

	// BLOCK_HAVE_DATA : full block available in blk*.dat
	BLOCK_HAVE_DATA uint32 = 8

	BLOCK_HAVE_UNDO uint32 = 16
	BLOCK_HAVE_MASK        = BLOCK_HAVE_DATA | BLOCK_HAVE_UNDO

	// BLOCK_FAILED_VALID : stage after last reached validness failed
	BLOCK_FAILED_VALID uint32 = 32
	// BLOCK_FAILED_CHILD : descends from failed block
	BLOCK_FAILED_CHILD uint32 = 64
	BLOCK_FAILED_MASK  uint32 = BLOCK_FAILED_VALID | BLOCK_FAILED_CHILD
)
