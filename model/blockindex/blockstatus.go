package blockindex

const (
	// BlockValidUnknown : Unused.
	BlockValidUnknown uint32 = 0

	// BlockValidHeader : parsed , version ok , hash satisfies claimed PoW, 1 <= vtx count <=max
	// timestamp not in future
	BlockValidHeader uint32 = 1

	// BlockValidTree : All parent headers found, difficulty matches, timestamp>= median
	// previous , checkpoint , Implies all parents are also at least TREE
	BlockValidTree uint32 = 2

	// BlockValidTransactions : Only first tx is coinBase, 2 <= coinBase input script length <= 100,
	// transactions valid, no duplicate txIds , sigOps , size , merkle root .
	// Implies all parents are at least TREE but not necessarily TRANSACTIONS.
	// When all parent blocks also have TRANSACTIONS , CBlockIndex ::nChainTx wll be set
	BlockValidTransactions uint32 = 3

	// BlockValidChain : outputs do not overspend inputs , no double spends , coinBase output ok
	// no immature coinBase spends , BIP30.
	// Implies all parents are also at least CHAIN.
	BlockValidChain uint32 = 4

	// BlockValidScripts : Scripts & Signatures ok. Implies all parents are also at least SCRIPTS.
	BlockValidScripts uint32 = 5

	// BlockValidMask : All validity bits
	// BlockValidMask = BlockValidHeader |
	//	BlockValidTree |
	//	BlockValidTransactions |
	//	BlockValidChain |
	//	BlockValidScripts
	BlockValidityMask uint32 = 0x07

	// BlockHaveData : full block available in blk*.dat
	BlockHaveData uint32 = 8
	// Undo data available in rev*.dat
	BlockHaveUndo uint32 = 16

	// The block is invalid.
	BlockFailed uint32 = 32
	// The block has an invalid parent.
	BlockFailedParent uint32 = 64
	// Mask used to check if the block failed.
	BlockInvalidMask = BlockFailed | BlockFailedParent
)
