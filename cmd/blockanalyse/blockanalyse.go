package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/tx"
	"os"
)

func main() {
	flag := 0
	flag |= os.O_RDONLY
	filePath := "/Users/freedom/project/other/test/data/000000000000000007a4afe23f9c4681ea37bb73b6070587e1503a08a4681e23.block"
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Error("Unable to state file %s\n", err)
		panic("Unable to state file ======")
	}
	file, err := os.OpenFile(filePath, flag, os.ModePerm)
	if file == nil || err != nil {
		log.Error("Unable to state file %s\n", err)
		panic("Unable to state file ======")
	}
	defer file.Close()
	fileSize := fileInfo.Size()
	srcBuf := make([]byte, fileSize)
	readSize, err := file.Read(srcBuf)
	if int64(readSize) != fileSize {
		log.Error("Unable to read file %s\n", err)
		panic("Unable to read file ======")
	}
	dstLen := readSize / 2
	dstBuf := make([]byte, dstLen)

	decodeLen, err := hex.Decode(dstBuf, srcBuf)
	if decodeLen != dstLen {
		log.Error("Unable to decode block %s\n", err)
		panic("Unable to decode block ======")
	}

	pblock := block.NewBlock()
	blockBuf := bytes.NewBuffer(dstBuf)

	err = pblock.Unserialize(blockBuf)
	if err != nil {
		log.Error("block unserialize err %s\n", err)
		panic("Unable to unserialize block ======")
	}
	inputs, outputs := getBlockInputsOutputs(pblock)
	blockHash := pblock.GetHash()

	fmt.Printf("blockhash: %s, block size: %d bytes, intputs: %d, outputs: %d, tx count: %d", blockHash.String(),
		(readSize-1)/2, inputs, outputs, len(pblock.Txs))

	return
}

func getBlockInputsOutputs(pblock *block.Block) (inputs, outputs int) {
	for _, ptx := range pblock.Txs {
		tempInputs, tempOutputs := getTxInputsOutputs(ptx)
		inputs += tempInputs
		outputs += tempOutputs
	}

	return
}
func getTxInputsOutputs(tx *tx.Tx) (inputs int, outputs int) {
	return len(tx.GetIns()), len(tx.GetOuts())
}
