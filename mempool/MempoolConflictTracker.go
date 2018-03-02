package mempool

import "github.com/btcboost/copernicus/model"

type MemPoolConflictRemovalTracker struct {
	mpool         *Mempool
	conflictedTxs []*model.Tx
}

func NewMempoolConflictRemoveTrack(pool *Mempool) *MemPoolConflictRemovalTracker {
	m := new(MemPoolConflictRemovalTracker)
	m.mpool = pool
	m.conflictedTxs = make([]*model.Tx, 0)
	//todo !!! register signal.
	return m
}

func (m *MemPoolConflictRemovalTracker) NotifyEntryRemoved(txRemove *model.Tx, reason int) {
	if reason == CONFLICT {
		m.conflictedTxs = append(m.conflictedTxs, txRemove)
	}
}

func (m *MemPoolConflictRemovalTracker) DelMempoolConflictRemoveTrack() {
	//todo !!! 注册信号，发送信号
}
