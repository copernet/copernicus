package p2p

import (
	"github.com/btccom/copernicus/model"
	"sync"
	"time"
)

type BlockProgressLogger struct {
	receivedLogBlocks int64
	receivedLogTx     int64
	LastBlockLogTime  time.Time
	progressAction    string
	lock              sync.Mutex
}

func newBlockProgressLogger(progressMessage string) *BlockProgressLogger {
	blockProgressLogger := BlockProgressLogger{
		LastBlockLogTime: time.Now(),
		progressAction:   progressMessage,
	}
	return &blockProgressLogger
}

func (blockLog *BlockProgressLogger) LogBlockHeight(block *model.Block) {
	blockLog.lock.Lock()
	defer blockLog.lock.Unlock()
	blockLog.receivedLogBlocks++
	blockLog.receivedLogTx += int64(len(block.Transactions))
	now := time.Now()
	duration := now.Sub(blockLog.LastBlockLogTime)
	if duration < time.Second*10 {
		return
	}
	durationMillis := int64(duration / time.Millisecond)
	timeDuration := 10 * time.Millisecond * time.Duration(durationMillis/10)
	blockStr := "blocks"
	if blockLog.receivedLogBlocks == 1 {
		blockStr = "block"
	}
	txStr := "transactions"
	if blockLog.receivedLogTx == 1 {
		txStr = "transxation"
	}
	log.Info("%s %d %s in the last %s (%d %s, height %d, %s)",
		blockLog.progressAction,
		blockLog.receivedLogBlocks,
		blockStr,
		timeDuration,
		blockLog.receivedLogTx,
		txStr, block.Height, block.BlockTime)
	blockLog.receivedLogBlocks = 0
	blockLog.receivedLogTx = 0
	blockLog.LastBlockLogTime = now

}
