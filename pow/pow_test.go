package pow

import (
	"math/big"
	"testing"

	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
)

func TestPowCalculateNextWorkRequired(t *testing.T) {
	nLastRetargetTime := int64(1261130161) // Block #30240
	var pindexLast blockchain.BlockIndex
	pindexLast.Height = 32255
	pindexLast.Time = 1262152739 // Block #32255
	pindexLast.Bits = 0x1d00ffff

	pow := Pow{}
	work := pow.CalculateNextWorkRequired(&pindexLast, nLastRetargetTime, msg.ActiveNetParams)
	if work != 0x1d00d86a {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00d86a, work)
		return
	}

	nLastRetargetTime = 1231006505
	pindexLast.Height = 2015
	pindexLast.Time = 1233061996
	pindexLast.Bits = 0x1d00ffff
	work = pow.CalculateNextWorkRequired(&pindexLast, nLastRetargetTime, msg.ActiveNetParams)
	if work != 0x1d00ffff {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00ffff, work)
		return
	}

	nLastRetargetTime = 1279008237
	pindexLast.Height = 68543
	pindexLast.Time = 1279297671
	pindexLast.Bits = 0x1c05a3f4
	work = pow.CalculateNextWorkRequired(&pindexLast, nLastRetargetTime, msg.ActiveNetParams)
	if work != 0x1c0168fd {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1c0168fd, work)
		return
	}

	nLastRetargetTime = 1263163443
	pindexLast.Height = 46367
	pindexLast.Time = 1269211443
	pindexLast.Bits = 0x1c387f6f
	work = pow.CalculateNextWorkRequired(&pindexLast, nLastRetargetTime, msg.ActiveNetParams)
	if work != 0x1d00e1fd {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00e1fd, work)
		return
	}
}

func getBlockIndex(pindexPrev *blockchain.BlockIndex, timeInterval int64, bits uint32) *blockchain.BlockIndex {
	block := new(blockchain.BlockIndex)
	block.PPrev = pindexPrev
	block.Height = pindexPrev.Height + 1
	block.Time = pindexPrev.Time + uint32(timeInterval)
	block.Bits = bits
	block.ChainWork = *big.NewInt(0).Add(&pindexPrev.ChainWork, blockchain.GetBlockProof(block))
	return block
}

func TestPowGetNextWorkRequired(t *testing.T) {
	blocks := make([]*blockchain.BlockIndex, 115)
	currentPow := big.NewInt(0).Rsh(msg.ActiveNetParams.PowLimit, 1)
	initialBits := blockchain.BigToCompact(currentPow)
	pow := Pow{}

	// Genesis block.
	blocks[0] = new(blockchain.BlockIndex)
	blocks[0].SetNull()
	blocks[0].Height = 0
	blocks[0].Time = 1269211443
	blocks[0].Bits = initialBits
	blocks[0].ChainWork = *blockchain.GetBlockProof(blocks[0])

	// Pile up some blocks.
	for i := 1; i < 100; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], int64(msg.ActiveNetParams.TargetTimePerBlock), initialBits)
	}

	blkHeaderDummy := model.BlockHeader{}
	// We start getting 2h blocks time. For the first 5 blocks, it doesn't
	// matter as the MTP is not affected. For the next 5 block, MTP difference
	// increases but stays below 12h.
	for i := 100; i < 110; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], 2*3600, initialBits)

		acValue := pow.GetNextWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		if acValue != initialBits {
			t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
				initialBits, acValue)
			return
		}

	}

	// Now we expect the difficulty to decrease.
	blocks[110] = getBlockIndex(blocks[109], 2*3600, initialBits)
	currentPow = blockchain.CompactToBig(blockchain.BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	acValue := pow.GetNextWorkRequired(blocks[110], &blkHeaderDummy, msg.ActiveNetParams)
	if acValue != blockchain.BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			blockchain.BigToCompact(currentPow), acValue)
		return
	}

	// As we continue with 2h blocks, difficulty continue to decrease.
	blocks[111] = getBlockIndex(blocks[110], 2*3600, blockchain.BigToCompact(currentPow))
	currentPow = blockchain.CompactToBig(blockchain.BigToCompact(currentPow))
	currentPow.Add(currentPow, new(big.Int).Rsh(currentPow, 2))
	acValue = pow.GetNextWorkRequired(blocks[111], &blkHeaderDummy, msg.ActiveNetParams)
	if acValue != blockchain.BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			blockchain.BigToCompact(currentPow), acValue)
		return
	}

	// We decrease again.
	blocks[112] = getBlockIndex(blocks[111], 2*3600, blockchain.BigToCompact(currentPow))
	currentPow = blockchain.CompactToBig(blockchain.BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	acValue = pow.GetNextWorkRequired(blocks[112], &blkHeaderDummy, msg.ActiveNetParams)
	if acValue != blockchain.BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			blockchain.BigToCompact(currentPow), acValue)
		return
	}

	// We check that we do not go below the minimal difficulty.
	blocks[113] = getBlockIndex(blocks[112], 2*3600, blockchain.BigToCompact(currentPow))
	currentPow = blockchain.CompactToBig(blockchain.BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	if blockchain.BigToCompact(msg.ActiveNetParams.PowLimit) == blockchain.BigToCompact(currentPow) {
		t.Errorf("the two value should not equal ")
		return
	}
	acValue = pow.GetNextWorkRequired(blocks[113], &blkHeaderDummy, msg.ActiveNetParams)
	if acValue != blockchain.BigToCompact(msg.ActiveNetParams.PowLimit) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			blockchain.BigToCompact(msg.ActiveNetParams.PowLimit), acValue)
		return
	}

	// Once we reached the minimal difficulty, we stick with it.
	blocks[114] = getBlockIndex(blocks[113], 2*3600, blockchain.BigToCompact(currentPow))
	if blockchain.BigToCompact(msg.ActiveNetParams.PowLimit) == blockchain.BigToCompact(currentPow) {
		t.Errorf("the two value should not equal ")
		return
	}
	acValue = pow.GetNextWorkRequired(blocks[114], &blkHeaderDummy, msg.ActiveNetParams)
	if acValue != blockchain.BigToCompact(msg.ActiveNetParams.PowLimit) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			blockchain.BigToCompact(msg.ActiveNetParams.PowLimit), acValue)
		return
	}
}

func TestPowGetNextCashWorkRequired(t *testing.T) {
	blocks := make([]*blockchain.BlockIndex, 3000)
	currentPow := big.NewInt(0).Rsh(msg.ActiveNetParams.PowLimit, 4)
	initialBits := blockchain.BigToCompact(currentPow)

	// Genesis block.
	blocks[0] = new(blockchain.BlockIndex)
	blocks[0].SetNull()
	blocks[0].Height = 0
	blocks[0].Time = 1269211443
	blocks[0].Bits = initialBits
	blocks[0].ChainWork = *blockchain.GetBlockProof(blocks[0])

	// Block counter.
	i := 0

	// Pile up some blocks every 10 mins to establish some history.
	for i = 1; i < 2050; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], 600, initialBits)
	}

	blkHeaderDummy := model.BlockHeader{}
	pow := Pow{}
	bits := pow.GetNextCashWorkRequired(blocks[2049], &blkHeaderDummy, msg.ActiveNetParams)

	// Difficulty stays the same as long as we produce a block every 10 mins.
	for j := 0; j < 10; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 600, bits)
		work := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		if work != bits {
			t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
				bits, work)
			return
		}
		i++
	}

	// Make sure we skip over blocks that are out of wack. To do so, we produce
	// a block that is far in the future, and then produce a block with the
	// expected timestamp.
	blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
	work := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
	if work != bits {
		t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
			bits, work)
		return
	}
	i++
	blocks[i] = getBlockIndex(blocks[i-1], 2*600-6000, bits)
	work = pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
	if work != bits {
		t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
			bits, work)
		return
	}
	i++

	// The system should continue unaffected by the block with a bogous
	// timestamps.
	for j := 0; j < 20; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 600, bits)
		work = pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		if work != bits {
			t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
				bits, work)
			return
		}
		i++
	}

	// We start emitting blocks slightly faster. The first block has no impact.
	blocks[i] = getBlockIndex(blocks[i-1], 550, bits)
	work = pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
	if work != bits {
		t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
			bits, work)
		return
	}
	i++

	// Now we should see difficulty increase slowly.
	for j := 0; j < 10; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 550, bits)
		nextBits := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		currentTarget := blockchain.CompactToBig(bits)
		nextTarget := blockchain.CompactToBig(nextBits)
		if nextTarget.Cmp(currentTarget) >= 0 {
			t.Errorf("the next work : %d should less current work : %d",
				blockchain.BigToCompact(nextTarget), blockchain.BigToCompact(currentTarget))
			return
		}

		if big.NewInt(0).Sub(currentTarget, nextTarget).Cmp(big.NewInt(0).Div(currentTarget, big.NewInt(1024))) >= 0 {
			t.Errorf("currentTarget sub nextTarget value : %d should less currentTarget left move 10 bit late value : %d",
				big.NewInt(0).Sub(currentTarget, nextTarget), big.NewInt(0).Div(currentTarget, big.NewInt(1024)))
			return
		}
		bits = nextBits
		i++
	}

	// Check the actual value.
	if bits != 0x1c0fe7b1 {
		t.Errorf("the bits value : %d should equal 0x1c0fe7b1\n", bits)
		return
	}

	// If we dramatically shorten block production, difficulty increases faster.
	for j := 0; j < 20; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 10, bits)
		nextBits := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		currentTarget := blockchain.CompactToBig(bits)
		nextTarget := blockchain.CompactToBig(nextBits)
		// Make sure that difficulty increases faster.
		if nextTarget.Cmp(currentTarget) >= 0 {
			t.Errorf("the next work : %d should less current work : %d",
				blockchain.BigToCompact(nextTarget), blockchain.BigToCompact(currentTarget))
			return
		}
		if big.NewInt(0).Sub(currentTarget, nextTarget).Cmp(big.NewInt(0).Div(currentTarget, big.NewInt(16))) >= 0 {
			t.Errorf("currentTarget sub nextTarget value : %d should less currentTarget left move 10 bit late value : %d",
				big.NewInt(0).Sub(currentTarget, nextTarget), big.NewInt(0).Div(currentTarget, big.NewInt(16)))
			return
		}

		i++
		bits = nextBits
	}

	// Check the actual value.
	if bits != 0x1c0db19f {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c0db19f)
		return
	}

	// We start to emit blocks significantly slower. The first block has no
	// impact.
	blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
	bits = pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
	i++

	// Check the actual value.
	if bits != 0x1c0d9222 {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c0d9222)
		return
	}

	// If we dramatically slow down block production, difficulty decreases.
	for j := 0; j < 93; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
		nextBits := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		currentTarget := blockchain.CompactToBig(bits)
		nextTarget := blockchain.CompactToBig(nextBits)
		// Check the difficulty decreases.
		if nextTarget.Cmp(msg.ActiveNetParams.PowLimit) > 0 {
			t.Errorf("nextTarget value : %d should less or equal powLimit : %d",
				nextBits, blockchain.BigToCompact(msg.ActiveNetParams.PowLimit))
			return
		}
		if nextTarget.Cmp(currentTarget) <= 0 {
			t.Errorf("nextTarget value : %d should greater currentTarget value : %d",
				nextBits, bits)
			return
		}
		if big.NewInt(0).Sub(nextTarget, currentTarget).Cmp(big.NewInt(0).Rsh(currentTarget, 3)) >= 0 {
			t.Errorf("nextTarget sub currentTarget value : %d should less currentTarget left move 3 bit late value : %d",
				big.NewInt(0).Sub(nextTarget, currentTarget), big.NewInt(0).Rsh(currentTarget, 3))
			return
		}
		bits = nextBits
		i++
	}

	// Check the actual value.
	if bits != 0x1c2f13b9 {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c2f13b9)
		return
	}

	// Due to the window of time being bounded, next block's difficulty actually
	// gets harder.
	blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
	bits = pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
	i++
	if bits != 0x1c2ee9bf {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c2ee9bf)
		return
	}

	// And goes down again. It takes a while due to the window being bounded and
	// the skewed block causes 2 blocks to get out of the window.
	for j := 0; j < 192; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
		nextBits := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		currentTarget := blockchain.CompactToBig(bits)
		nextTarget := blockchain.CompactToBig(nextBits)

		// Check the difficulty decreases.
		if nextTarget.Cmp(msg.ActiveNetParams.PowLimit) > 0 {
			t.Errorf("nextTarget value : %d should less or equal powLimit : %d",
				nextBits, blockchain.BigToCompact(msg.ActiveNetParams.PowLimit))
			return
		}
		if nextTarget.Cmp(currentTarget) <= 0 {
			t.Errorf("nextTarget value : %d should greater currentTarget value : %d",
				nextBits, bits)
			return
		}
		if big.NewInt(0).Sub(nextTarget, currentTarget).Cmp(big.NewInt(0).Div(currentTarget, big.NewInt(8))) >= 0 {
			t.Errorf("nextTarget sub currentTarget value : %d should less currentTarget left move 3 bit late value : %d",
				big.NewInt(0).Sub(nextTarget, currentTarget), big.NewInt(0).Div(currentTarget, big.NewInt(8)))
			return
		}

		i++
		bits = nextBits
	}

	if bits != 0x1d00ffff {
		t.Errorf("the bits value : %d should equal 0x1c2ee9bf \n", bits)
		return
	}

	// Once the difficulty reached the minimum allowed level, it doesn't get any
	// easier.
	for j := 0; j < 5; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
		nextBits := pow.GetNextCashWorkRequired(blocks[i], &blkHeaderDummy, msg.ActiveNetParams)
		// Check the difficulty stays constant.
		if nextBits != blockchain.BigToCompact(msg.ActiveNetParams.PowLimit) {
			t.Errorf("the nextbits : %d should equal powLimit value : %d",
				nextBits, blockchain.BigToCompact(msg.ActiveNetParams.PowLimit))
			return
		}
		i++
		bits = nextBits
	}

}
