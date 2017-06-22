package algorithm

type Int64Sorter [] int64

func (sorter Int64Sorter) Len() int {
	return len(sorter)
}

func (sorter Int64Sorter) Swap(i, j int) {
	sorter[i], sorter[j] = sorter[j], sorter[i]
}
func (sorter Int64Sorter) Less(i, j int) bool {
	return sorter[i] < sorter[j]
}
