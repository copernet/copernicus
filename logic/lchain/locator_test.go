package lchain_test

import (
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLocateBlocks(t *testing.T) {
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)
	defer os.RemoveAll(testDir)
	defer cleanTestEnv()

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	hashes, err := generateDummyBlocks(pubKey, 57, 1000000, 0, nil)
	assert.Nil(t, err)

	tchain := chain.GetInstance()
	indexEnd := tchain.FindBlockIndex(hashes[40])
	locate := chain.GetInstance().GetLocator(indexEnd)

	actual := lchain.LocateBlocks(locate, tchain.GetIndex(0).GetBlockHash())
	for i, hash := range actual {
		assert.Equal(t, hashes[41+i].String(), hash.String())
	}

	indexEnd = tchain.FindBlockIndex(hashes[56])
	locate = chain.GetInstance().GetLocator(indexEnd)
	actual = lchain.LocateBlocks(locate, tchain.GetIndex(50).GetBlockHash())
	assert.Equal(t, []util.Hash{}, actual)
}

func TestLocateHeaders(t *testing.T) {
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)
	defer os.RemoveAll(testDir)
	defer cleanTestEnv()

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	hashes, err := generateDummyBlocks(pubKey, 57, 1000000, 0, nil)
	assert.Nil(t, err)

	tchain := chain.GetInstance()
	indexEnd := tchain.FindBlockIndex(hashes[40])
	locate := chain.GetInstance().GetLocator(indexEnd)

	actual := lchain.LocateHeaders(locate, tchain.GetIndex(0).GetBlockHash())
	for i, header := range actual {
		assert.Equal(t, hashes[41+i].String(), header.Hash.String())
	}

	indexEnd = tchain.FindBlockIndex(hashes[56])
	locate = chain.GetInstance().GetLocator(indexEnd)
	actual = lchain.LocateHeaders(locate, tchain.GetIndex(50).GetBlockHash())
	assert.Equal(t, []block.BlockHeader{}, actual)

	hashTest := util.GetRandHash()
	actual = lchain.LocateHeaders(chain.NewBlockLocator([]util.Hash{}), hashTest)
	assert.Equal(t, []block.BlockHeader{}, actual)
}
