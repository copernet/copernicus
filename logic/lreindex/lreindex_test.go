package lreindex

import (
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"os"
	"strings"
	"testing"
)

var (
	unitTestDataDirPath string
	testFilePos         block.DiskBlockPos
	blockNumsInTestFile int
	testFileName        string
)

func initTestEnv(t *testing.T) {
	args := []string{"--testnet", "--reindex"}
	conf.Cfg = conf.InitConfig(args)
	var err error
	unitTestDataDirPath, err = conf.SetUnitTestDataDir(conf.Cfg)
	t.Logf("test in temp dir: %s", unitTestDataDirPath)
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}
	defer os.RemoveAll(unitTestDataDirPath)

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

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

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	mempool.InitMempool()

	crypto.InitSecp256()

}

func initTestBlockFile() (testFileName string, err error) {

	blocks := make([][]byte, 0)
	blockStrs := []string{
		// block 0
		"1D 01 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 3B A3 ED FD 7A 7B 12 B2 7A C7 2C 3E 67 76 8F 61 7F C8 1B C3 88 8A 51 32 3A 9F B8 AA 4B 1E 5E 4A DA E5 49 4D FF FF 00 1D 1A A4 AE 18 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 4D 04 FF FF 00 1D 01 04 45 54 68 65 20 54 69 6D 65 73 20 30 33 2F 4A 61 6E 2F 32 30 30 39 20 43 68 61 6E 63 65 6C 6C 6F 72 20 6F 6E 20 62 72 69 6E 6B 20 6F 66 20 73 65 63 6F 6E 64 20 62 61 69 6C 6F 75 74 20 66 6F 72 20 62 61 6E 6B 73 FF FF FF FF 01 00 F2 05 2A 01 00 00 00 43 41 04 67 8A FD B0 FE 55 48 27 19 67 F1 A6 71 30 B7 10 5C D6 A8 28 E0 39 09 A6 79 62 E0 EA 1F 61 DE B6 49 F6 BC 3F 4C EF 38 C4 F3 55 04 E5 1E C1 12 DE 5C 38 4D F7 BA 0B 8D 57 8A 4C 70 2B 6B F1 1D 5F AC 00 00 00 00",
		// block 1
		"BE 00 00 00 01 00 00 00 43 49 7F D7 F8 26 95 71 08 F4 A3 0F D9 CE C3 AE BA 79 97 20 84 E9 0E AD 01 EA 33 09 00 00 00 00 BA C8 B0 FA 92 7C 0A C8 23 42 87 E3 3C 5F 74 D3 8D 35 48 20 E2 47 56 AD 70 9D 70 38 FC 5F 31 F0 20 E7 49 4D FF FF 00 1D 03 E4 B6 72 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 0E 04 20 E7 49 4D 01 7F 06 2F 50 32 53 48 2F FF FF FF FF 01 00 F2 05 2A 01 00 00 00 23 21 02 1A EA F2 F8 63 8A 12 9A 31 56 FB E7 E5 EF 63 52 26 B0 BA FD 49 5F F0 3A FE 2C 84 3D 7E 3A 4B 51 AC 00 00 00 00",
		// block 2
		"BE 00 00 00 01 00 00 00 06 12 8E 87 BE 8B 1B 4D EA 47 A7 24 7D 55 28 D2 70 2C 96 82 6C 7A 64 84 97 E7 73 B8 00 00 00 00 E2 41 35 2E 3B EC 0A 95 A6 21 7E 10 C3 AB B5 4A DF A0 5A BB 12 C1 26 69 55 95 58 0F B9 2E 22 20 32 E7 49 4D FF FF 00 1D 00 D2 35 34 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 0E 04 32 E7 49 4D 01 0E 06 2F 50 32 53 48 2F FF FF FF FF 01 00 F2 05 2A 01 00 00 00 23 21 03 8A 7F 6E F1 C8 CA 0C 58 8A A5 3F A8 60 12 80 77 C9 E6 C1 1E 68 30 F4 D7 EE 4E 76 3A 56 B7 71 8F AC 00 00 00 00",
		// block 3
		"BE 00 00 00 01 00 00 00 20 78 2A 00 52 55 B6 57 69 6E A0 57 D5 B9 8F 34 DE FC F7 51 96 F6 4F 6E EA C8 02 6C 00 00 00 00 41 BA 5A FC 53 2A AE 03 15 1B 8A A8 7B 65 E1 59 4F 97 50 4A 76 8E 01 0C 98 C0 AD D7 92 16 24 71 86 E7 49 4D FF FF 00 1D 05 8D C2 B6 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 0E 04 86 E7 49 4D 01 51 06 2F 50 32 53 48 2F FF FF FF FF 01 00 F2 05 2A 01 00 00 00 23 21 03 F6 D9 FF 4C 12 95 94 45 CA 55 49 C8 11 68 3B F9 C8 8E 63 7B 22 2D D2 E0 31 11 54 C4 C8 5C F4 23 AC 00 00 00 00",
		// block 4
		"BE 00 00 00 01 00 00 00 10 BE FD C1 6D 28 1E 40 EC EC 65 B7 C9 97 6D DC 8F D9 BC 97 52 DA 58 27 27 6E 89 8B 00 00 00 00 4C 97 6D 57 76 DD A2 DA 30 D9 6E E8 10 CD 97 D2 3B A8 52 41 49 90 D6 4C 4C 72 0F 97 7E 65 1F 2D AA E7 49 4D FF FF 00 1D 02 A9 76 40 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 0E 04 AA E7 49 4D 01 1D 06 2F 50 32 53 48 2F FF FF FF FF 01 00 F2 05 2A 01 00 00 00 23 21 02 1F 72 DE 1C FF 17 77 A9 58 4F 31 AD C4 58 04 18 14 C3 BC 39 C6 62 41 AC 4D 43 13 6D 71 06 AE BE AC 00 00 00 00",
		// block 5
		"BE 00 00 00 01 00 00 00 DD E5 B6 48 F5 94 FD D2 EC 1C 40 83 76 2D D1 3B 19 7B B1 38 1E 74 B1 FF F9 0A 5D 8B 00 00 00 00 B3 C6 C6 C1 11 8C 3B 6A BA A1 7C 5A A7 4E E2 79 08 9A D3 4D C3 CE C3 64 05 22 73 75 41 CB 01 68 18 E8 49 4D FF FF 00 1D 02 DA 84 C0 01 01 00 00 00 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 FF FF FF FF 0E 04 18 E8 49 4D 01 6A 06 2F 50 32 53 48 2F FF FF FF FF 01 00 F2 05 2A 01 00 00 00 23 21 03 73 EA 73 9A 7E 74 DE EE B4 43 1A 6D 77 DF 41 97 2C 18 5E 6E 83 E1 C3 00 EB 91 31 09 63 39 83 E7 AC 00 00 00 00",
	}

	util.Shuffle(blockStrs)

	blockNumsInTestFile = len(blockStrs)

	for _, strData := range blockStrs {
		hexStr := strings.Replace(strData, " ", "", -1)
		blockBytes, _ := hex.DecodeString(hexStr)
		blocks = append(blocks, blockBytes)
	}

	testFilePos = block.DiskBlockPos{File: 0, Pos: 0}
	blkFile := disk.OpenBlockFile(&testFilePos, false)
	defer blkFile.Close()

	for _, data := range blocks {
		_, err := blkFile.Write(data)
		if err != nil {
			return "", err
		}
	}

	return disk.GetBlockPosFilename(testFilePos, "blk"), nil
}

func TestLoadExternalBlockFile(t *testing.T) {
	initTestEnv(t)
	var err error
	testFileName, err = initTestBlockFile()
	if err != nil {
		t.Errorf("init test block file failed: %s", err)
	}
	defer os.RemoveAll(testFileName)

	nLoaded, err := loadExternalBlockFile(testFileName, &testFilePos)
	if err != nil {
		if err.Error() == "EOF" {
			t.Logf("Access EOF when loading file, maybe have loaded all data")

		} else {
			t.Errorf("load External Block file <%s> failed: %s", testFileName, err)
		}
	}
	if nLoaded != blockNumsInTestFile {
		t.Errorf("should load %d blocks, but load %d blocks", blockNumsInTestFile, nLoaded)
	}
}

func TestReindex(t *testing.T) {
	initTestEnv(t)
	UnloadBlockIndex()
	var err error
	testFileName, err = initTestBlockFile()
	if err != nil {
		t.Errorf("init test block file failed: %s", err)
	}
	defer os.RemoveAll(testFileName)
	Reindex()
}

func TestUnloadBlockIndex(t *testing.T) {
	UnloadBlockIndex()
}
