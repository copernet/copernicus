package lblock

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
)

func init() {
	if conf.Cfg == nil {
		conf.Cfg = conf.InitConfig([]string{})
		model.SetTestNetParams()
		chain.InitGlobalChain()
	}
}

func initTestEnv(t *testing.T) (dirpath string, err error) {
	conf.Cfg = conf.InitConfig([]string{})

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)
	t.Logf("test in temp dir: %s", unitTestDataDirPath)
	if err != nil {
		return "", err
	}

	model.SetTestNetParams()

	persist.InitPersistGlobal()

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	chain.InitGlobalChain()

	lblockindex.LoadBlockIndexDB()

	mempool.InitMempool()

	crypto.InitSecp256()

	return unitTestDataDirPath, nil
}

func clearTestEnv(dirpath string) {
	os.RemoveAll(dirpath)
}

func getBlock(blockstr string) *block.Block {
	blk := block.NewBlock()
	blkBuf, _ := hex.DecodeString(blockstr)
	err := blk.Unserialize(bytes.NewReader(blkBuf))
	if err != nil {
		fmt.Printf("ut buf err:%v", err)
	} else {
		fmt.Printf("ut hash:%v", blk.Header.GetHash())
	}
	return blk
}

func TestGetBlockByIndex(t *testing.T) {
	blHeader := block.NewBlockHeader()
	blHeader.Version = 1
	blHeader.HashPrevBlock = mainNetGenesisHash
	blHeader.MerkleRoot = mainNetGenesisMerkleRoot
	blHeader.Time = 1231469665
	blHeader.Bits = 0x1d00ffff
	blHeader.Nonce = 2573394689
	blIndex := blockindex.NewBlockIndex(blHeader)
	blIndex.Height = 1

	if _, ret := GetBlockByIndex(blIndex, chain.GetInstance().GetParams()); ret == nil {
		t.Errorf("TestGetBlockByIndex test 1 check not exist block failed")
	}
}

func TestAcceptBlockHeader(t *testing.T) {
	dirpath, err := initTestEnv(t)
	defer clearTestEnv(dirpath)
	if err != nil {
		return
	}
	var badHeader = &block.BlockHeader{
		Version:       0x20000000,
		HashPrevBlock: getHash("0000000000000000006a65508da81986b4e95e731e4f36d93b388cb89301ddbc"),
		MerkleRoot:    getHash("559c09b5e1d6fc86c01d84ef648e8c72331f06a2bcd38c8509bccfe0a5fcab52"),
		Time:          1539397926,
		Bits:          0x1802295b,
		Nonce:         1836649123,
	}
	if blkIdx, err := AcceptBlockHeader(badHeader); err == nil {
		t.Errorf("TestAcceptBlockHeader test 1 failed. index:%s",
			blkIdx.GetBlockHash().String())
	}

	blk1Str := "0100000006128E87BE8B1B4DEA47A7247D5528D2702C96826C7A648497E773B800000000E241352E3" +
		"BEC0A95A6217E10C3ABB54ADFA05ABB12C126695595580FB92E222032E7494DFFFF001D00D23534010100000" +
		"0010000000000000000000000000000000000000000000000000000000000000000FFFFFFFF0E0432E7494D0" +
		"10E062F503253482FFFFFFFFF0100F2052A010000002321038A7F6EF1C8CA0C588AA53FA860128077C9E6C11" +
		"E6830F4D7EE4E763A56B7718FAC00000000"
	blk1 := getBlock(blk1Str)
	blk2Str := "0100000020782A005255B657696EA057D5B98F34DEFCF75196F64F6EEAC8026C0000000041BA5AFC5" +
		"32AAE03151B8AA87B65E1594F97504A768E010C98C0ADD79216247186E7494DFFFF001D058DC2B6010100000" +
		"0010000000000000000000000000000000000000000000000000000000000000000FFFFFFFF0E0486E7494D0" +
		"151062F503253482FFFFFFFFF0100F2052A01000000232103F6D9FF4C12959445CA5549C811683BF9C88E637" +
		"B222DD2E0311154C4C85CF423AC00000000"
	blk2 := getBlock(blk2Str)
	blk1Index := blockindex.NewBlockIndex(&blk1.Header)
	blk1Index.Status = blockindex.BlockFailed
	chain.GetInstance().AddToIndexMap(blk1Index)
	if blkIdx, err := AcceptBlockHeader(&blk1.Header); err == nil {
		t.Errorf("TestAcceptBlockHeader test 2 failed. index:%s", blkIdx.GetBlockHash().String())
	}
	blk1Index.Status = blockindex.BlockValidMask
	chain.GetInstance().AddToIndexMap(blk1Index)
	if _, err := AcceptBlockHeader(&blk2.Header); err != nil {
		t.Errorf("TestAcceptBlockHeader test 3 failed. err:%v", err)
	}
}

func TestAcceptBlock(t *testing.T) {
	dirpath, err := initTestEnv(t)
	defer clearTestEnv(dirpath)
	if err != nil {
		return
	}

	fRequested := false
	fNewBlock := false

	genesisBlk := chain.GetInstance().GetParams().GenesisBlock
	blkIdx, pos, err := AcceptBlock(genesisBlk, fRequested, nil, &fNewBlock)
	if err != nil || blkIdx.Height != 0 || blkIdx.Prev != nil || pos == nil {
		t.Errorf("TestAcceptBlock test 1 check genesis block failed. error:%v", err)
	}
	genesisBlkIndex := blockindex.NewBlockIndex(&genesisBlk.Header)
	chain.GetInstance().SetTip(genesisBlkIndex)

	firstBlockStr := "0100000043497FD7F826957108F4A30FD9CEC3AEBA79972084E90EAD01EA330900000000BAC8B0FA" +
		"927C0AC8234287E33C5F74D38D354820E24756AD709D7038FC5F31F020E7494DFFFF001D03E4B6720101000000010000000000" +
		"000000000000000000000000000000000000000000000000000000FFFFFFFF0E0420E7494D017F062F503253482FFFFFFFFF01" +
		"00F2052A010000002321021AEAF2F8638A129A3156FBE7E5EF635226B0BAFD495FF03AFE2C843D7E3A4B51AC00000000"
	firstBlk := getBlock(firstBlockStr)
	blkIdx2, pos2, err := AcceptBlock(firstBlk, fRequested, nil, &fNewBlock)
	if err != nil || blkIdx2.Height != 1 || *blkIdx2.Prev.GetBlockHash() != genesisBlk.Header.Hash || pos2 == nil {
		t.Errorf("TestAcceptBlock test 2 check valid block failed. error:%v", err)
	}

	blkIdxRepeat, _, err := AcceptBlock(firstBlk, fRequested, nil, &fNewBlock)
	if err != nil || blkIdx2.Height != blkIdxRepeat.Height {
		t.Error("TestAcceptBlock test 3 check repeat block fail")
	}

	blankBlk := block.NewBlock()
	_, _, err = AcceptBlock(blankBlk, fRequested, nil, &fNewBlock)
	if err == nil {
		t.Errorf("TestAcceptBlock test 4 check blank block failed")
	}

	// test valid orphan block
	orphanBlockPrevStr := "0100000006128E87BE8B1B4DEA47A7247D5528D2702C96826C7A648497E773B800000000E241352E3" +
		"BEC0A95A6217E10C3ABB54ADFA05ABB12C126695595580FB92E222032E7494DFFFF001D00D23534010100000" +
		"0010000000000000000000000000000000000000000000000000000000000000000FFFFFFFF0E0432E7494D0" +
		"10E062F503253482FFFFFFFFF0100F2052A010000002321038A7F6EF1C8CA0C588AA53FA860128077C9E6C11" +
		"E6830F4D7EE4E763A56B7718FAC00000000"
	orphanBlockPrev := getBlock(orphanBlockPrevStr)
	orphanBlockStr := "0100000020782A005255B657696EA057D5B98F34DEFCF75196F64F6EEAC8026C0000000041BA5AFC5" +
		"32AAE03151B8AA87B65E1594F97504A768E010C98C0ADD79216247186E7494DFFFF001D058DC2B6010100000" +
		"0010000000000000000000000000000000000000000000000000000000000000000FFFFFFFF0E0486E7494D0" +
		"151062F503253482FFFFFFFFF0100F2052A01000000232103F6D9FF4C12959445CA5549C811683BF9C88E637" +
		"B222DD2E0311154C4C85CF423AC00000000"
	orphanBlock := getBlock(orphanBlockStr)
	orphanBlockIndexPrev := blockindex.NewBlockIndex(&orphanBlockPrev.Header)
	orphanBlockIndexPrev.Status = blockindex.BlockFailed
	chain.GetInstance().AddToIndexMap(orphanBlockIndexPrev)
	_, _, err = AcceptBlock(orphanBlock, fRequested, nil, &fNewBlock)
	if err == nil {
		t.Errorf("TestAcceptBlock test 5 failed. check invalid orphan block error:%v", err)
	}
	orphanBlockIndexPrev.Status = blockindex.BlockValidMask
	chain.GetInstance().AddToIndexMap(orphanBlockIndexPrev)
	_, _, err = AcceptBlock(orphanBlock, fRequested, nil, &fNewBlock)
	if err != nil {
		t.Errorf("TestAcceptBlock test 6 failed. check valid orphan block error:%v", err)
	}
	orphanBlockPrev.Txs[0].UpdateInScript(0, script.NewEmptyScript())
	_, _, err = AcceptBlock(orphanBlockPrev, fRequested, pos, &fNewBlock)
	if err == nil {
		t.Errorf("TestAcceptBlock test 7 failed. check invalid orphan block")
	}

	orphanBlockInvalid := block.NewBlock()
	orphanBlockInvalid.Header = block.BlockHeader{
		Version:       0x20000000,
		HashPrevBlock: getHash("0000000000000000006a65508da81986b4e95e731e4f36d93b388cb89301ddbc"),
		MerkleRoot:    getHash("559c09b5e1d6fc86c01d84ef648e8c72331f06a2bcd38c8509bccfe0a5fcab52"),
		Time:          1539397926,
		Bits:          0x1802295b,
		Nonce:         1836649339,
	}
	_, _, err = AcceptBlock(orphanBlockInvalid, fRequested, nil, &fNewBlock)
	if err == nil {
		t.Errorf("TestAcceptBlock test 8 failed. check invalid orphan block error")
	}
}

func TestGetBlockScriptFlags(t *testing.T) {
	header := &chain.GetInstance().GetParams().GenesisBlock.Header
	blkIndex := blockindex.NewBlockIndex(header)
	ret := GetBlockScriptFlags(blkIndex)
	if ret != 0 {
		t.Errorf("TestGetBlockScriptFlags test failed. ret:%v", ret)
	}
}

func TestGetBlockSubsidy(t *testing.T) {
	amt := GetBlockSubsidy(0, chain.GetInstance().GetParams())
	if amt.ToBTC() != 50 {
		t.Errorf("TestGetBlockSubsidy test failed. amount:%v", amt.ToBTC())
	}
}

func TestContextualCheckBlock(t *testing.T) {
	err := ContextualCheckBlock(chain.GetInstance().GetParams().GenesisBlock, nil)
	if err != nil {
		t.Errorf("TestContextualCheckBlock test 1 failed. error:%v", err)
	}

	firstBlockStr := "0100000043497FD7F826957108F4A30FD9CEC3AEBA79972084E90EAD01EA330900000000BAC8B0FA" +
		"927C0AC8234287E33C5F74D38D354820E24756AD709D7038FC5F31F020E7494DFFFF001D03E4B6720101000000010000000000" +
		"000000000000000000000000000000000000000000000000000000FFFFFFFF0E0420E7494D017F062F503253482FFFFFFFFF01" +
		"00F2052A010000002321021AEAF2F8638A129A3156FBE7E5EF635226B0BAFD495FF03AFE2C843D7E3A4B51AC00000000"
	firstBlk := getBlock(firstBlockStr)
	blkIndex := blockindex.NewBlockIndex(&chain.GetInstance().GetParams().GenesisBlock.Header)
	err = ContextualCheckBlock(firstBlk, blkIndex)
	if err != nil {
		t.Errorf("TestContextualCheckBlock test 2 failed.")
	}
}
