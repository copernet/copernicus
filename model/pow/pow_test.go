package pow

import (
	"testing"

	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/util"
	"math"
	"math/big"
)

func TestPowCalculateNextWorkRequired(t *testing.T) {
	model.ActiveNetParams = &model.MainNetParams

	lastRetargetTime := int64(1261130161) // Block #30240
	var indexLast blockindex.BlockIndex
	indexLast.Height = 32255
	indexLast.Header.Time = 1262152739 // Block #32255
	indexLast.Header.Bits = 0x1d00ffff

	pow := Pow{}
	work := pow.calculateNextWorkRequired(&indexLast, lastRetargetTime, model.ActiveNetParams)
	if work != 0x1d00d86a {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00d86a, work)
		return
	}

	lastRetargetTime = 1231006505
	indexLast.Height = 2015
	indexLast.Header.Time = 1233061996
	indexLast.Header.Bits = 0x1d00ffff
	work = pow.calculateNextWorkRequired(&indexLast, lastRetargetTime, model.ActiveNetParams)
	if work != 0x1d00ffff {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00ffff, work)
		return
	}

	lastRetargetTime = 1279008237
	indexLast.Height = 68543
	indexLast.Header.Time = 1279297671
	indexLast.Header.Bits = 0x1c05a3f4
	work = pow.calculateNextWorkRequired(&indexLast, lastRetargetTime, model.ActiveNetParams)
	if work != 0x1c0168fd {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1c0168fd, work)
		return
	}

	lastRetargetTime = 1263163443
	indexLast.Height = 46367
	indexLast.Header.Time = 1269211443
	indexLast.Header.Bits = 0x1c387f6f
	work = pow.calculateNextWorkRequired(&indexLast, lastRetargetTime, model.ActiveNetParams)
	if work != 0x1d00e1fd {
		t.Errorf("expect the next work : %d, but actual work is %d ", 0x1d00e1fd, work)
		return
	}
}

func getBlockIndex(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	block := new(blockindex.BlockIndex)
	block.Prev = indexPrev
	block.Height = indexPrev.Height + 1
	block.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	block.Header.Bits = bits
	block.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, GetBlockProof(block))
	return block
}

func TestPowGetNextWorkRequired(t *testing.T) {
	blocks := make([]*blockindex.BlockIndex, 115)
	currentPow := big.NewInt(0).Rsh(model.ActiveNetParams.PowLimit, 1)
	initialBits := BigToCompact(currentPow)
	pow := Pow{}

	// Genesis block.
	blocks[0] = new(blockindex.BlockIndex)
	blocks[0].SetNull()
	blocks[0].Height = 0
	blocks[0].Header.Time = 1269211443
	blocks[0].Header.Bits = initialBits
	blocks[0].ChainWork = *GetBlockProof(blocks[0])

	blkHeaderDummy := block.BlockHeader{}

	//TestNet3Params EDA will not happened
	if model.ActiveNetParams.Name == model.TestNetParams.Name {
		var i int
		for i = 1; i < 10; i++ {
			blocks[i] = getBlockIndex(blocks[i-1], int64(model.ActiveNetParams.TargetTimePerBlock), initialBits)
			acValue := pow.GetNextWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
			if acValue != initialBits {
				t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
					initialBits, acValue)
				return
			}
		}
		blocks[i] = getBlockIndex(blocks[i-1], 2*600+1, initialBits)
		acValue := pow.GetNextWorkRequired(blocks[i-1], blocks[i].GetBlockHeader(), model.ActiveNetParams)
		limitBits := BigToCompact(model.ActiveNetParams.PowLimit)
		if acValue != limitBits {
			t.Errorf("the two value should be equal, but expect value : %x, actual value : %x",
				limitBits, acValue)
			return
		}
		return
	}

	// Pile up some blocks.
	for i := 1; i < 100; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], int64(model.ActiveNetParams.TargetTimePerBlock), initialBits)
	}
	// We start getting 2h blocks time. For the first 5 blocks, it doesn't
	// matter as the MTP is not affected. For the next 5 block, MTP difference
	// increases but stays below 12h.
	for i := 100; i < 110; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], 2*3600, initialBits)

		acValue := pow.GetNextWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		if acValue != initialBits {
			t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
				initialBits, acValue)
			return
		}

	}

	// Now we expect the difficulty to decrease.
	blocks[110] = getBlockIndex(blocks[109], 2*3600, initialBits)
	currentPow = CompactToBig(BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	acValue := pow.GetNextWorkRequired(blocks[110], &blkHeaderDummy, model.ActiveNetParams)
	if acValue != BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			BigToCompact(currentPow), acValue)
		return
	}

	// As we continue with 2h blocks, difficulty continue to decrease.
	blocks[111] = getBlockIndex(blocks[110], 2*3600, BigToCompact(currentPow))
	currentPow = CompactToBig(BigToCompact(currentPow))
	currentPow.Add(currentPow, new(big.Int).Rsh(currentPow, 2))
	acValue = pow.GetNextWorkRequired(blocks[111], &blkHeaderDummy, model.ActiveNetParams)
	if acValue != BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			BigToCompact(currentPow), acValue)
		return
	}

	// We decrease again.
	blocks[112] = getBlockIndex(blocks[111], 2*3600, BigToCompact(currentPow))
	currentPow = CompactToBig(BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	acValue = pow.GetNextWorkRequired(blocks[112], &blkHeaderDummy, model.ActiveNetParams)
	if acValue != BigToCompact(currentPow) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			BigToCompact(currentPow), acValue)
		return
	}

	// We check that we do not go below the minimal difficulty.
	blocks[113] = getBlockIndex(blocks[112], 2*3600, BigToCompact(currentPow))
	currentPow = CompactToBig(BigToCompact(currentPow))
	currentPow.Add(currentPow, big.NewInt(0).Rsh(currentPow, 2))
	if BigToCompact(model.ActiveNetParams.PowLimit) == BigToCompact(currentPow) {
		t.Errorf("the two value should not equal ")
		return
	}
	acValue = pow.GetNextWorkRequired(blocks[113], &blkHeaderDummy, model.ActiveNetParams)
	if acValue != BigToCompact(model.ActiveNetParams.PowLimit) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			BigToCompact(model.ActiveNetParams.PowLimit), acValue)
		return
	}

	// Once we reached the minimal difficulty, we stick with it.
	blocks[114] = getBlockIndex(blocks[113], 2*3600, BigToCompact(currentPow))
	if BigToCompact(model.ActiveNetParams.PowLimit) == BigToCompact(currentPow) {
		t.Errorf("the two value should not equal ")
		return
	}
	acValue = pow.GetNextWorkRequired(blocks[114], &blkHeaderDummy, model.ActiveNetParams)
	if acValue != BigToCompact(model.ActiveNetParams.PowLimit) {
		t.Errorf("the two value should be equal, but expect value : %d, actual value : %d",
			BigToCompact(model.ActiveNetParams.PowLimit), acValue)
		return
	}
}

func TestPowGetNextCashWorkRequired(t *testing.T) {
	blocks := make([]*blockindex.BlockIndex, 3000)
	currentPow := big.NewInt(0).Rsh(model.ActiveNetParams.PowLimit, 4)
	initialBits := BigToCompact(currentPow)

	// Genesis block.
	blocks[0] = new(blockindex.BlockIndex)
	blocks[0].SetNull()
	blocks[0].Height = 0
	blocks[0].Header.Time = 1269211443
	blocks[0].Header.Bits = initialBits
	blocks[0].ChainWork = *GetBlockProof(blocks[0])

	// Block counter.
	var i int
	// Pile up some blocks every 10 mins to establish some history.
	for i = 1; i < 2050; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], 600, initialBits)
	}

	blkHeaderDummy := block.BlockHeader{}
	pow := Pow{}
	bits := pow.getNextCashWorkRequired(blocks[2049], &blkHeaderDummy, model.ActiveNetParams)

	// Difficulty stays the same as long as we produce a block every 10 mins.
	for j := 0; j < 10; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 600, bits)
		work := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
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
	work := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
	if work != bits {
		t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
			bits, work)
		return
	}
	i++
	blocks[i] = getBlockIndex(blocks[i-1], 2*600-6000, bits)
	work = pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
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
		work = pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		if work != bits {
			t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
				bits, work)
			return
		}
		i++
	}

	// We start emitting blocks slightly faster. The first block has no impact.
	blocks[i] = getBlockIndex(blocks[i-1], 550, bits)
	work = pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
	if work != bits {
		t.Errorf("the two value should equal, but the expect velue : %d, the actual value : %d",
			bits, work)
		return
	}
	i++

	// Now we should see difficulty increase slowly.
	for j := 0; j < 10; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 550, bits)
		nextBits := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		currentTarget := CompactToBig(bits)
		nextTarget := CompactToBig(nextBits)
		if nextTarget.Cmp(currentTarget) >= 0 {
			t.Errorf("the next work : %d should less current work : %d",
				BigToCompact(nextTarget), BigToCompact(currentTarget))
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
		nextBits := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		currentTarget := CompactToBig(bits)
		nextTarget := CompactToBig(nextBits)
		// Make sure that difficulty increases faster.
		if nextTarget.Cmp(currentTarget) >= 0 {
			t.Errorf("the next work : %d should less current work : %d",
				BigToCompact(nextTarget), BigToCompact(currentTarget))
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
	bits = pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
	i++

	// Check the actual value.
	if bits != 0x1c0d9222 {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c0d9222)
		return
	}

	// If we dramatically slow down block production, difficulty decreases.
	for j := 0; j < 93; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
		nextBits := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		currentTarget := CompactToBig(bits)
		nextTarget := CompactToBig(nextBits)
		// Check the difficulty decreases.
		if nextTarget.Cmp(model.ActiveNetParams.PowLimit) > 0 {
			t.Errorf("nextTarget value : %d should less or equal powLimit : %d",
				nextBits, BigToCompact(model.ActiveNetParams.PowLimit))
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
	bits = pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
	i++
	if bits != 0x1c2ee9bf {
		t.Errorf("the bits value : %d should equal %d \n", bits, 0x1c2ee9bf)
		return
	}

	// And goes down again. It takes a while due to the window being bounded and
	// the skewed block causes 2 blocks to get out of the window.
	for j := 0; j < 192; j++ {
		blocks[i] = getBlockIndex(blocks[i-1], 6000, bits)
		nextBits := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		currentTarget := CompactToBig(bits)
		nextTarget := CompactToBig(nextBits)

		// Check the difficulty decreases.
		if nextTarget.Cmp(model.ActiveNetParams.PowLimit) > 0 {
			t.Errorf("nextTarget value : %d should less or equal powLimit : %d",
				nextBits, BigToCompact(model.ActiveNetParams.PowLimit))
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
		nextBits := pow.getNextCashWorkRequired(blocks[i], &blkHeaderDummy, model.ActiveNetParams)
		// Check the difficulty stays constant.
		if nextBits != BigToCompact(model.ActiveNetParams.PowLimit) {
			t.Errorf("the nextbits : %d should equal powLimit value : %d",
				nextBits, BigToCompact(model.ActiveNetParams.PowLimit))
			return
		}
		i++
		bits = nextBits
	}

	b := new(big.Int)
	by, err := hex.DecodeString("000000000000000000000000000000000000000000000028803b6c018c06d7c5")
	if err != nil {
		panic(err)
	}
	c := b.SetBytes(by)
	fmt.Println("height : 1188696, chainwork : ", c.String())

	c, e := c.SetString("000000000000000000000000000000000000000000000028803b6c018c06d7c5", 16)
	if !e {
		panic(e)
	}
	fmt.Println("height : 1188696, chainwork : ", c.String())

}

func TestPow_CheckProofOfWork(t *testing.T) {

	hash := util.HashFromString("0000000000000000000740e6d6defb455a045d87a4b05a77f84df382a0e6e16b")
	pow := Pow{}
	if ok := pow.CheckProofOfWork(hash, 0x172c0da7, model.ActiveNetParams); !ok {
		t.Errorf("CheckProofOfWork should pass")
	}

	if ok := pow.CheckProofOfWork(hash, 0x1d00ffff, model.ActiveNetParams); !ok {
		t.Errorf("CheckProofOfWork should pass")
	}

	if ok := pow.CheckProofOfWork(hash, 0, model.ActiveNetParams); ok {
		t.Errorf("CheckProofOfWork should not pass")
	}

	if ok := pow.CheckProofOfWork(hash, uint32(math.MaxUint32), model.ActiveNetParams); ok {
		t.Errorf("CheckProofOfWork should not pass")
	}
}

func TestPow_GetNextWorkRequired(t *testing.T) {
	pow := Pow{}

	exp := uint32(0x1D00FFFF)
	actual := pow.GetNextWorkRequired(nil, nil, model.ActiveNetParams)
	if actual != exp {
		t.Errorf("GetNextWorkRequired Error,check exp = 0x%x, actual = 0x%x", exp, actual)
	}
}
