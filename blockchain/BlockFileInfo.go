package blockchain

type BlockFileInfo struct {
	Blocks      int
	Size        int
	UndoSize    int
	HeightFirst int
	HeightLast  int
	timeFirst   uint64
	timeLast    uint64
}
