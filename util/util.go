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

func MaxI32(a, b int32) int32 {
	if a < b {
		return b
	}
	return a
}

func MinI32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func MaxU32(a, b uint32) uint32 {
	if a < b {
		return b
	}
	return a
}

func MinU32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}