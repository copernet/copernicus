package mempool

import (
	"unsafe"

	"github.com/btcboost/copernicus/model"
	"gopkg.in/fatih/set.v0"
)

// RecursiveDynamicUsage todo need to dynamically calculate the memory footprint of the object , haven't figured out
// how to calculate it better yet , How about the size  of the object's serialization
func RecursiveDynamicUsage(tx *model.Tx) int {
	return tx.SerializeSize()
}

func DynamicUsage(Item interface{}) int {
	size := 0
	switch typeOf := Item.(type) {
	case *set.Set:
		num := typeOf.Size()
		switch v := typeOf.List()[0].(type) {
		case TxMempoolEntry:
			size = num * int(unsafe.Sizeof(v))
		case *TxMempoolEntry:
			size = num * int(unsafe.Sizeof(v))
		case int:
			size = num * int(unsafe.Sizeof(v))

		}
	case TxMempoolEntry:
		size = int(unsafe.Sizeof(typeOf))
	case *TxMempoolEntry:
		size = int(unsafe.Sizeof(typeOf))

	}
	return size
}

/*MallocUsage Compute the memory used for dynamically allocated but owned data structures.
 * For generic data types, this is *not* recursive.
 * DynamicUsage(vector<vector<int>>) will compute the memory used for the
 * vector<int>'s, but not for the ints inside. This is for efficiency reasons,
 * as these functions are intended to be fast. If application data structures
 * require more accurate inner accounting, they should iterate themselves, or
 * use more efficient caching + updating on modification.
 */
func MallocUsage(alloc int) int {
	tmp := new(int)
	// Measured on libc6 2.19 on Linux
	if alloc == 0 {
		return 0
	} else if unsafe.Sizeof(tmp) == 8 {
		return ((alloc + 31) >> 4) << 4
	} else if unsafe.Sizeof(tmp) == 4 {
		return ((alloc + 15) >> 3) << 3
	}
	panic("the platForm is not supported !!!")

}
