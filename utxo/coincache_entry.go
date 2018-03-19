package utxo

const (
	// CoinEntryDirty : This cache entry is potentially different from the version in the
	// parent view.
	CoinEntryDirty = 1 << 0
	// CoinEntryFresh : The parent view does not have this entry (or it is pruned).
	CoinEntryFresh = 1 << 1
)

type CoinsCacheEntry struct {
	Coin  *Coin
	Flags uint8
}

func NewCoinsCacheEntry(coin *Coin) *CoinsCacheEntry {
	coinsCacheEntry := CoinsCacheEntry{Flags: 0, Coin: coin}
	return &coinsCacheEntry
}
