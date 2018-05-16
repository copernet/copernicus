package util

func MaxI(a, b int64) int64 {
	if a < b {
		return b
	}
	return a
}

func MinI(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxU(a, b uint64) uint64 {
	if a < b {
		return b
	}
	return a
}

func MinU(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
