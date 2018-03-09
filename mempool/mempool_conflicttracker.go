package mempool

import "github.com/btcboost/copernicus/core"

type PoolConflictRemovalTracker struct {
	mpool         *Mempool
	conflictedTxs []*core.Tx
}

func NewMempoolConflictRemoveTrack(pool *Mempool) *PoolConflictRemovalTracker {
	m := new(PoolConflictRemovalTracker)
	m.mpool = pool
	m.conflictedTxs = make([]*core.Tx, 0)
	//todo !!! register signal.
	return m
}

func (m *PoolConflictRemovalTracker) NotifyEntryRemoved(txRemove *core.Tx, reason int) {
	if reason == CONFLICT {
		m.conflictedTxs = append(m.conflictedTxs, txRemove)
	}
}

func (m *PoolConflictRemovalTracker) DelMempoolConflictRemoveTrack() {
	//todo !!! 注册信号，发送信号
}
