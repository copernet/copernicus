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
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

const (
	// testnet, height=1
	blk1str = "0100000043497FD7F826957108F4A30FD9CEC3AEBA79972084E90EAD01EA330900000000" +
		"BAC8B0FA927C0AC8234287E33C5F74D38D354820E24756AD709D7038FC5F31F020E7494DFFFF00" +
		"1D03E4B67201010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0420E7494D017F062F503253482FFFFFFFFF0100F2052A01000000232102" +
		"1AEAF2F8638A129A3156FBE7E5EF635226B0BAFD495FF03AFE2C843D7E3A4B51AC00000000"
	// testnet, height=2
	blk2str = "0100000006128E87BE8B1B4DEA47A7247D5528D2702C96826C7A648497E773B800000000" +
		"E241352E3BEC0A95A6217E10C3ABB54ADFA05ABB12C126695595580FB92E222032E7494DFFFF00" +
		"1D00D2353401010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0432E7494D010E062F503253482FFFFFFFFF0100F2052A01000000232103" +
		"8A7F6EF1C8CA0C588AA53FA860128077C9E6C11E6830F4D7EE4E763A56B7718FAC00000000"
	// testnet, height=3
	blk3str = "0100000020782A005255B657696EA057D5B98F34DEFCF75196F64F6EEAC8026C00000000" +
		"41BA5AFC532AAE03151B8AA87B65E1594F97504A768E010C98C0ADD79216247186E7494DFFFF00" +
		"1D058DC2B601010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0486E7494D0151062F503253482FFFFFFFFF0100F2052A01000000232103" +
		"F6D9FF4C12959445CA5549C811683BF9C88E637B222DD2E0311154C4C85CF423AC00000000"
	// testnet, height=4
	blk4str = "0100000010BEFDC16D281E40ECEC65B7C9976DDC8FD9BC9752DA5827276E898B00000000" +
		"4C976D5776DDA2DA30D96EE810CD97D23BA852414990D64C4C720F977E651F2DAAE7494DFFFF00" +
		"1D02A9764001010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E04AAE7494D011D062F503253482FFFFFFFFF0100F2052A01000000232102" +
		"1F72DE1CFF1777A9584F31ADC458041814C3BC39C66241AC4D43136D7106AEBEAC00000000"
	// testnet, height=5
	blk5str = "01000000DDE5B648F594FDD2EC1C4083762DD13B197BB1381E74B1FFF90A5D8B00000000" +
		"B3C6C6C1118C3B6ABAA17C5AA74EE279089AD34DC3CEC3640522737541CB016818E8494DFFFF00" +
		"1D02DA84C001010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0418E8494D016A062F503253482FFFFFFFFF0100F2052A01000000232103" +
		"73EA739A7E74DEEEB4431A6D77DF41972C185E6E83E1C300EB913109633983E7AC00000000"
	// testnet, height=1200000
	chkPointBlkHeaderStr = "00000020b817a3c806dfe735f65202294a708246872e113cd277dd8d016" +
		"1bff800000000218443d1bbda9d43e9917ef205341d748cac8257871336ab99b4ee395d5ab7d0a" +
		"8184a5affff001ddeb8f6aa"
)

func TestMain(m *testing.M) {
	workDir, err := initTestEnv()
	if err == nil {
		fmt.Printf("test in temp dir: %s\n", workDir)
		m.Run()
	} else {
		fmt.Printf("init test env error:%s\n", err.Error())
	}
	if len(workDir) > 0 {
		os.RemoveAll(workDir)
		fmt.Println("clear test env")
	}
}

func initTestEnv() (string, error) {
	conf.Cfg = conf.InitConfig([]string{})

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)
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

func clearIndexMap() {
	indexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	chain.GetInstance().InitLoad(indexMap, branch)
	chain.GetInstance().ClearActive()
}

func getBlock(blockstr string) *block.Block {
	blk := block.NewBlock()
	blkBuf, _ := hex.DecodeString(blockstr)
	err := blk.Unserialize(bytes.NewReader(blkBuf))
	if err != nil {
		return nil
	}
	return blk
}

func TestGetBlockByIndex(t *testing.T) {
	blkIdx := blockindex.NewBlockIndex(&chain.GetInstance().GetParams().GenesisBlock.Header)
	// empty now
	if _, err := GetBlockByIndex(blkIdx, chain.GetInstance().GetParams()); err == nil {
		t.Errorf("TestGetBlockByIndex test 1 check not exist block failed")
	}
}

func TestAcceptBlockHeader(t *testing.T) {
	clearIndexMap()

	genesisHeader := &chain.GetInstance().GetParams().GenesisBlock.Header
	if _, err := AcceptBlockHeader(genesisHeader); err != nil {
		t.Errorf("TestAcceptBlockHeader test 1 check genesis header failed")
	}

	blk1 := getBlock(blk1str)
	badHeader := &blk1.Header
	badHeader.Nonce = 12345
	if blkIdx, err := AcceptBlockHeader(badHeader); err == nil {
		t.Errorf("TestAcceptBlockHeader test 2 check invalid header failed. index:%s",
			blkIdx.GetBlockHash().String())
	}

	blk2 := getBlock(blk2str)
	blk2Index := blockindex.NewBlockIndex(&blk2.Header)
	blk2Index.Status = blockindex.BlockFailed
	chain.GetInstance().AddToIndexMap(blk2Index)
	if _, err := AcceptBlockHeader(&blk2.Header); err == nil {
		t.Errorf("TestAcceptBlockHeader test 3 check invalid status header failed")
	}

	blk2Index.Status = blockindex.BlockValidMask
	chain.GetInstance().AddToIndexMap(blk2Index)
	if _, err := AcceptBlockHeader(&blk2.Header); err != nil {
		t.Errorf("TestAcceptBlockHeader test 4 check valid header failed. err:%v", err)
	}

	blk5 := getBlock(blk5str)
	if _, err := AcceptBlockHeader(&blk5.Header); err == nil {
		t.Errorf("TestAcceptBlockHeader test 5 check orphan header failed")
	}

	// add check point
	chkPointBlkHeader := block.NewBlockHeader()
	chkBuf, _ := hex.DecodeString(chkPointBlkHeaderStr)
	chkPointBlkHeader.Unserialize(bytes.NewReader(chkBuf))
	chkPointIndex := blockindex.NewBlockIndex(chkPointBlkHeader)
	chain.GetInstance().AddToIndexMap(chkPointIndex)

	blk3 := getBlock(blk3str)
	if _, err := AcceptBlockHeader(&blk3.Header); err == nil {
		t.Errorf("TestAcceptBlockHeader test 6 check checkpoint height failed")
	}
}

func TestAcceptBlock(t *testing.T) {
	clearIndexMap()

	fRequested := false
	fNewBlock := false

	genesisBlk := chain.GetInstance().GetParams().GenesisBlock
	genesisBlkIdx, pos, err := AcceptBlock(genesisBlk, fRequested, nil, &fNewBlock)
	if err != nil || genesisBlkIdx.Height != 0 || genesisBlkIdx.Prev != nil || pos == nil {
		t.Errorf("TestAcceptBlock test 1 check genesis block failed. error:%v", err)
	}
	genesisBlkIndex := blockindex.NewBlockIndex(&genesisBlk.Header)
	chain.GetInstance().SetTip(genesisBlkIndex)

	blk1 := getBlock(blk1str)
	blk1Idx, pos2, err := AcceptBlock(blk1, fRequested, pos, &fNewBlock)
	if err != nil || blk1Idx.Height != 1 || *blk1Idx.Prev.GetBlockHash() != genesisBlk.Header.Hash || pos2 == nil {
		t.Errorf("TestAcceptBlock test 2 check valid block failed. error:%v", err)
	}

	blk1IdxRepeat, _, err := AcceptBlock(blk1, fRequested, pos2, &fNewBlock)
	if err != nil || blk1IdxRepeat.Height != blk1IdxRepeat.Height {
		t.Error("TestAcceptBlock test 3 check repeat block fail")
	}

	blankBlk := block.NewBlock()
	if _, _, err = AcceptBlock(blankBlk, fRequested, pos2, &fNewBlock); err == nil {
		t.Errorf("TestAcceptBlock test 4 check blank block failed")
	}

	blk2 := getBlock(blk2str)
	blk3 := getBlock(blk3str)
	blk2Idx := blockindex.NewBlockIndex(&blk2.Header)
	blk2Idx.Status = blockindex.BlockFailed
	chain.GetInstance().AddToIndexMap(blk2Idx)
	if _, _, err = AcceptBlock(blk3, fRequested, nil, &fNewBlock); err == nil {
		t.Errorf("TestAcceptBlock test 5 failed. check block whose prev index invalid")
	}

	blk2Idx.Status = blockindex.BlockValidMask
	chain.GetInstance().AddToIndexMap(blk2Idx)
	if _, _, err = AcceptBlock(blk3, fRequested, nil, &fNewBlock); err != nil {
		t.Errorf("TestAcceptBlock test 6 failed. check valid orphan block error:%v", err)
	}

	blk2.Txs[0].UpdateInScript(0, script.NewEmptyScript())
	if _, _, err = AcceptBlock(blk2, fRequested, pos, &fNewBlock); err == nil {
		t.Errorf("TestAcceptBlock test 7 failed. check block whose merkle root invalid")
	}

	blk5 := getBlock(blk5str)
	blk5.Header.Nonce = 12345
	if _, _, err = AcceptBlock(blk5, fRequested, nil, &fNewBlock); err == nil {
		t.Errorf("TestAcceptBlock test 8 failed. check orphan block whose header error")
	}
}

func TestCheckBlock(t *testing.T) {
	blk1 := getBlock(blk1str)
	blk1.Checked = true
	blk1.Header.Nonce = 12345
	blk1.Txs[0].UpdateInScript(0, script.NewEmptyScript())
	if err := CheckBlock(blk1, true, true); err != nil {
		t.Errorf("TestCheckBlock test 1 check checked status failed")
	}

	blk2 := getBlock(blk2str)
	blk2.Header.Nonce = 12345
	if err := CheckBlock(blk2, true, false); err == nil {
		t.Errorf("TestCheckBlock test 2 check bad header failed")
	}

	blk3 := getBlock(blk3str)
	blk3.Txs[0].UpdateInScript(0, script.NewEmptyScript())
	if err := CheckBlock(blk3, false, true); err == nil {
		t.Errorf("TestCheckBlock test 3 check bad merkle root failed")
	}

	blk4 := getBlock(blk4str)
	if err := CheckBlock(blk4, true, true); err != nil {
		t.Errorf("TestCheckBlock test 4 check valid block failed")
	}

	blk5 := getBlock(blk5str)
	nMaxBlockSize := consensus.DefaultMaxBlockSize
	emptyTx := tx.NewEmptyTx()
	repeat := nMaxBlockSize / int(emptyTx.EncodeSize())
	for i := 0; i < repeat; i++ {
		blk5.Txs = append(blk5.Txs, emptyTx)
	}
	if err := CheckBlock(blk5, false, false); err == nil {
		t.Errorf("TestCheckBlock test 5 check big block by tx number failed")
	}

	blk5 = getBlock(blk5str)
	txn := blk5.Txs[0]
	repeat = nMaxBlockSize / int(txn.EncodeSize())
	for i := 0; i < repeat; i++ {
		blk5.Txs = append(blk5.Txs, txn)
	}
	if err := CheckBlock(blk5, false, false); err == nil {
		t.Errorf("TestCheckBlock test 6 check big block failed. block size:%v", blk5.EncodeSize())
	}

	genesisBlock := chain.GetInstance().GetParams().GenesisBlock
	if err := CheckBlock(genesisBlock, true, true); err != nil {
		t.Errorf("TestCheckBlock test 7 check genesis block failed")
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
	amt := model.GetBlockSubsidy(0, chain.GetInstance().GetParams())
	if amt.ToBTC() != 50 {
		t.Errorf("TestGetBlockSubsidy test 1 check height=0 failed. amount:%v", amt.ToBTC())
	}

	height := chain.GetInstance().GetParams().SubsidyReductionInterval
	amt = model.GetBlockSubsidy(height, chain.GetInstance().GetParams())
	if amt.ToBTC() != 25 {
		t.Errorf("TestGetBlockSubsidy test 2 check reduce half failed. amount:%v", amt.ToBTC())
	}

	height = chain.GetInstance().GetParams().SubsidyReductionInterval * 64
	amt = model.GetBlockSubsidy(height, chain.GetInstance().GetParams())
	if amt.ToBTC() != 0 {
		t.Errorf("TestGetBlockSubsidy test 3 check reduce to zero failed. amount:%v", amt.ToBTC())
	}
}

func TestContextualCheckBlock(t *testing.T) {
	if err := ContextualCheckBlock(chain.GetInstance().GetParams().GenesisBlock, nil); err != nil {
		t.Errorf("TestContextualCheckBlock test 1 check genesis block failed. error:%s", err.Error())
	}

	blk1 := getBlock(blk1str)
	blk0Index := blockindex.NewBlockIndex(&chain.GetInstance().GetParams().GenesisBlock.Header)
	if err := ContextualCheckBlock(blk1, blk0Index); err != nil {
		t.Errorf("TestContextualCheckBlock test 2 check valid block failed. error:%s", err.Error())
	}

	blk2 := getBlock(blk2str)
	blk1Index := blockindex.NewBlockIndex(&blk1.Header)
	txn := tx.NewGenesisCoinbaseTx()
	repeat := 8 * consensus.OneMegaByte / txn.SerializeSize()
	for i := uint32(0); i < repeat; i++ {
		blk2.Txs = append(blk2.Txs, txn)
	}
	if err := ContextualCheckBlock(blk2, blk1Index); err == nil {
		t.Errorf("TestContextualCheckBlock test 3 check big block failed")
	}

	blk1Index.Header.Time = uint32(chain.GetInstance().GetParams().MonolithActivationTime)
	if err := ContextualCheckBlock(blk2, blk1Index); err != nil {
		t.Errorf("TestContextualCheckBlock test 4 check MonolithActivationTime failed")
	}
}
