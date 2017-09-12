package mempool

import "github.com/btcboost/copernicus/model"

// RecursiveDynamicUsage todo need to dynamically calculate the memory footprint of the object , haven't figured out
// how to calculate it better yet , How about the size  of the object's serialization
func RecursiveDynamicUsage(tx *model.Tx) int {
	return tx.SerializeSize()
}
