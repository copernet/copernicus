package mempool

type CacheTxMap struct {
	m   map[*TxMempoolEntry]interface{}
	txs []*TxMempoolEntry
}

func (txMap *CacheTxMap) Len() int {
	return len(txMap.m)
}

func (txMap *CacheTxMap) Less(i, j int) bool {
	return txMap.txs[i].TxRef.Hash.Cmp(&txMap.txs[j].TxRef.Hash) > 0
}

func (txMap *CacheTxMap) Swap(i, j int) {
	txMap.txs[i], txMap.txs[j] = txMap.txs[j], txMap.txs[i]

}
