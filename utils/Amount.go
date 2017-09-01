package utils

const (
	COIN      int64 = 100000000
	CENT      int64 = 1000000
	MAX_MONEY int64 = 21000000 * COIN
)

func MoneyRange(value int64) bool {
	return value >= 0 && value <= MAX_MONEY
}
