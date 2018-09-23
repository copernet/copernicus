package model

import (
	"github.com/copernet/copernicus/model/block"
)

var GenesisBlock = block.NewGenesisBlock()
var GenesisBlockHash = GenesisBlock.GetHash()

var TestNetGenesisBlock = block.NewTestNetGenesisBlock()
var TestNetGenesisHash = TestNetGenesisBlock.GetHash()

var RegTestGenesisBlock = block.NewRegTestGenesisBlock()
var RegTestGenesisHash = RegTestGenesisBlock.GetHash()
