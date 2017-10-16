package utxo

const (
	// COIN_ENTRY_DIRTY : This cache entry is potentially different from the version in the
	// parent view.
	COIN_ENTRY_DIRTY = 1 << 0
	// COUB_ENTRY_FRESH : The parent view does not have this entry (or it is pruned).
	COUB_ENTRY_FRESH = 1 << 1
)

type CoinsCacheEntry struct {
	Coin  *Coin
	Flags uint8
}

func NewCoinsCacheEntry(coin *Coin) *CoinsCacheEntry {
	coinsCacheEntry := CoinsCacheEntry{Flags: 0, Coin: coin}
	return &coinsCacheEntry
}
