package policy

const (
	/*MAX_TX_SIGOPS_COUNT allowed number of signature check operations per transaction. */
	MAX_TX_SIGOPS_COUNT uint64 = 20000
	/*ONE_MEGABYTE 1MB */
	ONE_MEGABYTE uint64 = 1000000

	/*DEFAULT_MAX_GENERATED_BLOCK_SIZE Default for -blockmaxsize, which controls the maximum size of block the
	 * mining code will create **/
	DEFAULT_MAX_GENERATED_BLOCK_SIZE uint64 = 2 * ONE_MEGABYTE
	/*DEFAULT_BLOCK_PRIORITY_SIZE Default for -blockprioritysize, maximum space for zero/low-fee transactions*/
	DEFAULT_BLOCK_PRIORITY_SIZE uint64 = 0
	/*DEFAULT_BLOCK_MIN_TX_FEE Default for -blockmintxfee, which sets the minimum feerate for a transaction
	 * in blocks created by mining code **/
	DEFAULT_BLOCK_MIN_TX_FEE uint = 1000
	/*MAX_STANDARD_TX_SIZE The maximum size for transactions we're willing to relay/mine */
	MAX_STANDARD_TX_SIZE uint = 100000
	/*MAX_P2SH_SIGOPS Maximum number of signature check operations in an IsStandard() P2SH script*/
	MAX_P2SH_SIGOPS uint = 15
	/*MAX_STANDARD_TX_SIGOPS The maximum number of sigops we're willing to relay/mine in a single tx */
	MAX_STANDARD_TX_SIGOPS uint = uint(MAX_TX_SIGOPS_COUNT / 5)
	/*DEFAULT_MAX_MEMPOOL_SIZE Default for -maxmempool, maximum megabytes of mempool memory usage */
	DEFAULT_MAX_MEMPOOL_SIZE uint = 300
	/*DEFAULT_INCREMENTAL_RELAY_FEE Default for -incrementalrelayfee, which sets the minimum feerate increase
	 * for mempool limiting or BIP 125 replacement **/
	DEFAULT_INCREMENTAL_RELAY_FEE uint = 1000
	/*DEFAULT_BYTES_PER_SIGOP Default for -bytespersigop */
	DEFAULT_BYTES_PER_SIGOP uint = 20
	/*MAX_STANDARD_P2WSH_STACK_ITEMS The maximum number of witness stack items in a standard P2WSH script */
	MAX_STANDARD_P2WSH_STACK_ITEMS uint = 100
	/*MAX_STANDARD_P2WSH_STACK_ITEM_SIZE The maximum size of each witness stack item in a standard P2WSH script */
	MAX_STANDARD_P2WSH_STACK_ITEM_SIZE uint = 80
	/*MAX_STANDARD_P2WSH_SCRIPT_SIZE The maximum size of a standard witnessScript */
	MAX_STANDARD_P2WSH_SCRIPT_SIZE uint = 3600
)
