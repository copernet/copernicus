package msg

import (
	"math/big"
	"time"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

const ANTI_REPLAY_COMMITMENT = "Bitcoin: A Peer-to-Peer Electronic Cash System"

var ActiveNetParams = &MainNetParams

var (
	bigOne = big.NewInt(1)
	//2^224 -1
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)
	//2^255 -1
	regressingPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
	testNet3PowLimit   = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
	simNetPowlimit     = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 225), bigOne)
)

type DeploymentPos int

const (
	DEPLOYMENT_TESTDUMMY DeploymentPos = iota
	// DEPLOYMENT_CSV Deployment of BIP68, BIP112, and BIP113.
	DEPLOYMENT_CSV
	// MAX_VERSION_BITS_DEPLOYMENTS NOTE: Also add new deployments to VersionBitsDeploymentInfo in
	// versionbits.cpp
	MAX_VERSION_BITS_DEPLOYMENTS
)

type ChainTxData struct {
	Time    time.Time
	TxCount int64
	TxRate  float64
}

type BIP9Deployment struct {
	/** Bit position to select the particular bit in nVersion. */
	Bit int
	/** Start MedianTime for version bits miner confirmation. Can be a date in
	 * the past */
	StartTime int64
	/** Timeout/expiry MedianTime for the deployment attempt. */
	Timeout int64
}

type BitcoinParams struct {
	Name                         string
	BitcoinNet                   btcutil.BitcoinNet
	DefaultPort                  string
	DNSSeeds                     []btcutil.DNSSeed
	GenesisBlock                 *BlockMessage
	GenesisHash                  *utils.Hash
	PowLimit                     *big.Int
	PowLimitBits                 uint32
	CoinbaseMaturity             uint16
	SubsidyReductionInterval     int32
	TargetTimespan               time.Duration
	TargetTimePerBlock           time.Duration
	RetargetAdjustmentFactor     int64
	ReduceMinDifficulty          bool
	MinDiffReductionTime         time.Duration
	GenerateSupported            bool
	Checkpoints                  []*model.Checkpoint
	FPowNoRetargeting            bool
	FPowAllowMinDifficultyBlocks bool

	// Enforce current block version once network has
	// upgraded.  This is part of BIP0034.
	BlockEnforceNumRequired uint64

	// Reject previous block versions once network has
	// upgraded.  This is part of BIP0034.
	BlockRejectNumRequired uint64

	// The number of nodes to check.  This is part of BIP0034.
	BlockUpgradeNumToCheck uint64

	RequireStandard     bool
	RelayNonStdTxs      bool
	PubKeyHashAddressID byte
	ScriptHashAddressID byte
	PrivatekeyID        byte
	HDPrivateKeyID      [4]byte
	HDPublicKeyID       [4]byte
	HDCoinType          uint32

	//Block height and hash at which BIP34 becomes active
	BIP34Height int
	BIP34Hash   utils.Hash

	//Block height at which BIP65 becomes active
	BIP65Height int
	//Block height at which BIP66 becomes active
	BIP66Height int
	//Block height at which UAHF kicks in
	UAHFHeight int
	//Activation time at which the cash HF kicks in.
	CashHardForkActivationTime int64
	//Block height at which OP_RETURN replay protection stops
	AntiReplayOpReturnSunsetHeight int
	AntiReplayOpReturnCommitment   []byte
	RuleChangeActivationThreshold  uint32
	MinerConfirmationWindow        uint32
	Deployments                    [MAX_VERSION_BITS_DEPLOYMENTS]BIP9Deployment
	// The best chain should have at least this much work.
	MinimumChainWork big.Int
	// By default assume that the signatures in ancestors of this block are valid.
	DefaultAssumeValid big.Int
	PruneAfterHeight   int
	chainTxData        ChainTxData
}

func (param *BitcoinParams) TxData() *ChainTxData {
	return &param.chainTxData
}

var MainNetParams = BitcoinParams{
	Name:        "mainnet",
	BitcoinNet:  btcutil.MainNet,
	DefaultPort: "8333",
	DNSSeeds: []btcutil.DNSSeed{
		{Host: "seed.bitcoin.sipa.be", HasFiltering: true},  // Pieter Wuille
		{Host: "dnsseed.bluematt.me", HasFiltering: true},   // Matt Corallo
		{Host: "seed.bitcoinstats.com", HasFiltering: true}, // Chris Decker
		{Host: "bitseed.xf2.org", HasFiltering: true},
		{Host: "seed.bitcoinstats.com", HasFiltering: true},
		{Host: "seed.bitnodes.io", HasFiltering: false},
	},
	GenesisBlock:             &GenesisBlock,
	GenesisHash:              &GenesisHash,
	PowLimit:                 mainPowLimit,
	PowLimitBits:             GenesisBlock.Block.BlockHeader.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	BIP34Height:              227931,
	//BIP34Hash:                      utils.Hash{0x000000000000024b89b42a942fe0d9fea3bb44ab7bd1b19115dd6a759c0808b8},
	BIP65Height:                    388381,
	BIP66Height:                    363725,
	AntiReplayOpReturnSunsetHeight: 530000,
	RuleChangeActivationThreshold:  1916,
	MinerConfirmationWindow:        2016,
	AntiReplayOpReturnCommitment:   []byte(ANTI_REPLAY_COMMITMENT),
	Deployments: [MAX_VERSION_BITS_DEPLOYMENTS]BIP9Deployment{
		DEPLOYMENT_TESTDUMMY: {28, 1199145601, 1230767999},
		DEPLOYMENT_CSV:       {0, 1462060800, 1493596800},
	},
	TargetTimespan:           60 * 60 * 24 * 14,
	TargetTimePerBlock:       60 * 10,
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0,
	GenerateSupported:        false,
	Checkpoints: []*model.Checkpoint{
		{11111, utils.HashFromString("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
		{33333, utils.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")},
		{74000, utils.HashFromString("0000000000573993a3c9e41ce34471c079dcf5f52a0e824a81e7f953b8661a20")},
		{105000, utils.HashFromString("00000000000291ce28027faea320c8d2b054b2e0fe44a773f3eefb151d6bdc97")},
		{134444, utils.HashFromString("00000000000005b12ffd4cd315cd34ffd4a594f430ac814c91184a0d42d2b0fe")},
		{168000, utils.HashFromString("000000000000099e61ea72015e79632f216fe6cb33d7899acb35b75c8303b763")},
		{193000, utils.HashFromString("000000000000059f452a5f7340de6682a977387c17010ff6e6c3bd83ca8b1317")},
		{210000, utils.HashFromString("000000000000048b95347e83192f69cf0366076336c639f9b7228e9ba171342e")},
		{216116, utils.HashFromString("00000000000001b4f4b433e81ee46494af945cf96014816a4e2370f11b23df4e")},
		{225430, utils.HashFromString("00000000000001c108384350f74090433e7fcf79a606b8e797f065b130575932")},
		{250000, utils.HashFromString("000000000000003887df1f29024b06fc2200b55f8af8f35453d7be294df2d214")},
		{267300, utils.HashFromString("000000000000000a83fbd660e918f218bf37edd92b748ad940483c7c116179ac")},
		{279000, utils.HashFromString("0000000000000001ae8c72a0b0c301f67e3afca10e819efa9041e458e9bd7e40")},
		{300255, utils.HashFromString("0000000000000000162804527c6e9b9f0563a280525f9d08c12041def0a0f3b2")},
		{319400, utils.HashFromString("000000000000000021c6052e9becade189495d1c539aa37c58917305fd15f13b")},
		{343185, utils.HashFromString("0000000000000000072b8bf361d01a6ba7d445dd024203fafc78768ed4368554")},
		{352940, utils.HashFromString("000000000000000010755df42dba556bb72be6a32f3ce0b6941ce4430152c9ff")},
		{382320, utils.HashFromString("00000000000000000a8dc6ed5b133d0eb2fd6af56203e4159789b092defd8ab2")},
	},
	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	RelayNonStdTxs:      false,
	PubKeyHashAddressID: 0x00, // starts with 1
	ScriptHashAddressID: 0x05, // starts with 3
	PrivatekeyID:        0x80, // starts with 5 (uncompressed) or K (compressed)
	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub
	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 0,

	FPowNoRetargeting:          false,
	CashHardForkActivationTime: 1510600000,
	UAHFHeight:                 478559,
}

var RegressionNetParams = BitcoinParams{
	Name:                     "regtest",
	BitcoinNet:               btcutil.TestNet,
	DefaultPort:              "18444",
	DNSSeeds:                 []btcutil.DNSSeed{},
	GenesisBlock:             &RegressionTestGenesisBlock,
	GenesisHash:              &RegressionTestGenesisHash,
	PowLimit:                 regressingPowLimit,
	PowLimitBits:             RegressionTestGenesisBlock.Block.BlockHeader.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 150,
	TargetTimespan:           60 * 60 * 24 * 14,
	TargetTimePerBlock:       60 * 10,
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20,
	GenerateSupported:        true,
	Checkpoints:              nil,
	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	RelayNonStdTxs:      true,
	PubKeyHashAddressID: 0x6f, // starts with m or n
	ScriptHashAddressID: 0xc4, // starts with 2
	PrivatekeyID:        0xef, // starts with 9 (uncompressed) or c (compressed)
	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with xpub
	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1,
}

var TestNet3Params = BitcoinParams{
	Name:        "testnet3",
	BitcoinNet:  btcutil.TestNet3,
	DefaultPort: "18333",
	DNSSeeds: []btcutil.DNSSeed{
		{Host: "testnet-seed.bitcoin.schildbach.de", HasFiltering: false},
		{Host: "testnet-seed.bitcoin.petertodd.org", HasFiltering: true},
		{Host: "testnet-seed.bluematt.me", HasFiltering: false},
	},
	GenesisBlock:             &TestNet3GenesisBlock,
	GenesisHash:              &TestNet3GenesisHash,
	PowLimit:                 testNet3PowLimit,
	PowLimitBits:             GenesisBlock.Block.BlockHeader.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimespan:           60 * 60 * 24 * 14,
	TargetTimePerBlock:       60 * 10,
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20,
	GenerateSupported:        false,
	Checkpoints: []*model.Checkpoint{
		{546, utils.HashFromString("000000002a936ca763904c3c35fce2f3556c559c0214345d31b1bcebf76acb70")},
	},
	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	RelayNonStdTxs:      true,
	PubKeyHashAddressID: 0x6f, // starts with 1
	ScriptHashAddressID: 0xc4, // starts with 3
	PrivatekeyID:        0xef, // starts with 5 (uncompressed) or K (compressed)
	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with xpub
	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 1,
}

var SimNetParams = BitcoinParams{
	Name:                     "simnet",
	BitcoinNet:               btcutil.SimNet,
	DefaultPort:              "18555",
	DNSSeeds:                 []btcutil.DNSSeed{},
	GenesisBlock:             &SimNetGenesisBlock,
	GenesisHash:              &SimNetGenesisHash,
	PowLimit:                 simNetPowlimit,
	PowLimitBits:             SimNetGenesisBlock.Block.BlockHeader.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	TargetTimespan:           60 * 60 * 24 * 14,
	TargetTimePerBlock:       60 * 10,
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20,
	GenerateSupported:        false,
	Checkpoints:              nil,
	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	RelayNonStdTxs:      true,
	PubKeyHashAddressID: 0x3f, // starts with 1
	ScriptHashAddressID: 0x7b, // starts with 3
	PrivatekeyID:        0x64, // starts with 5 (uncompressed) or K (compressed)
	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0x04, 0x20, 0xb9, 0x00}, // starts with xprv
	HDPublicKeyID:  [4]byte{0x04, 0x20, 0xbd, 0x3a}, // starts with xpub
	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	HDCoinType: 115,
}

var (
	RegisteredNets          = make(map[btcutil.BitcoinNet]struct{})
	PubKeyHashAddressIDs    = make(map[byte]struct{})
	ScriptHashAddressIDs    = make(map[byte]struct{})
	HDPrivateToPublicKeyIDs = make(map[[4]byte][]byte)
)

func init() {
	mustRegister(&MainNetParams)
	mustRegister(&TestNet3Params)
	mustRegister(&RegressionNetParams)
	mustRegister(&SimNetParams)

}

func Register(bitcoinParams *BitcoinParams) error {
	if _, ok := RegisteredNets[bitcoinParams.BitcoinNet]; ok {
		return errors.New("duplicate bitcoin network")
	}
	RegisteredNets[bitcoinParams.BitcoinNet] = struct{}{}
	PubKeyHashAddressIDs[bitcoinParams.PubKeyHashAddressID] = struct{}{}
	ScriptHashAddressIDs[bitcoinParams.ScriptHashAddressID] = struct{}{}
	HDPrivateToPublicKeyIDs[bitcoinParams.HDPrivateKeyID] = bitcoinParams.HDPublicKeyID[:]
	return nil
}
func IsPublicKeyHashAddressID(id byte) bool {
	_, ok := PubKeyHashAddressIDs[id]
	return ok
}
func IsScriptHashAddressid(id byte) bool {
	_, ok := ScriptHashAddressIDs[id]
	return ok
}
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	if len(id) != 4 {
		return nil, errors.New("unknown hd private extended key bytes")
	}
	var key [4]byte
	copy(key[:], id)
	pubBytes, ok := HDPrivateToPublicKeyIDs[key]
	if !ok {
		return nil, errors.New("unknown hd private extended key bytes")

	}
	return pubBytes, nil
}
func mustRegister(bitcoinParams *BitcoinParams) {
	err := Register(bitcoinParams)
	if err != nil {
		panic("failed to register network :" + err.Error())
	}
	work, ok := big.NewInt(0).SetString("000000000000000000000000000000000000000000796b6d5908f8db26c3cf44", 16)
	if !ok {
		panic("error")
	}
	bitcoinParams.MinimumChainWork = *work
	work, ok = big.NewInt(0).SetString("000000000000000004694d6c74b532faf99fc072181f870bfb4a6c9930f7440c", 16)
	if !ok {
		panic("err")
	}
	bitcoinParams.DefaultAssumeValid = *work
}

func (param *BitcoinParams) DifficultyAdjustmentInterval() int64 {
	return int64(param.TargetTimespan / param.TargetTimePerBlock)
}
