package mempool

import "github.com/btcboost/copernicus/model"

type PoolConflictRemovalTracker struct {
	mpool         *Mempool
	conflictedTxs []*model.Tx
}

func NewMempoolConflictRemoveTrack(pool *Mempool) *PoolConflictRemovalTracker {
	m := new(PoolConflictRemovalTracker)
	m.mpool = pool
	m.conflictedTxs = make([]*model.Tx, 0)
	//todo !!! register signal.
	return m
}

func (m *PoolConflictRemovalTracker) NotifyEntryRemoved(txRemove *model.Tx, reason int) {
	if reason == CONFLICT {
		m.conflictedTxs = append(m.conflictedTxs, txRemove)
	}
}

func (m *PoolConflictRemovalTracker) DelMempoolConflictRemoveTrack() {
	//todo !!! 注册信号，发送信号
}
