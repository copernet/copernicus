package model

import (
	"errors"
	"github.com/copernet/copernicus/conf"
	"math/big"
	"time"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

const AntiReplayCommitment = "Bitcoin: A Peer-to-Peer Electronic Cash System"

var ActiveNetParams = &MainNetParams

var (
	bigOne = big.NewInt(1)
	// 2^224 -1
	mainPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)
	// 2^255 -1
	regressingPowLimit = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne)
	testNetPowLimit    = new(big.Int).Sub(new(big.Int).Lsh(bigOne, 224), bigOne)
)

type ChainTxData struct {
	Time    time.Time
	TxCount int64
	TxRate  float64
}

// DNSSeed identifies a DNS seed.
type DNSSeed struct {
	// Host defines the hostname of the seed.
	Host string

	// HasFiltering defines whether the seed supports filtering
	// by service flags (wire.ServiceFlag).
	HasFiltering bool
}

type BitcoinParams struct {
	consensus.Param
	Name                     string
	BitcoinNet               wire.BitcoinNet
	DiskMagic                wire.BitcoinNet
	DefaultPort              string
	DNSSeeds                 []DNSSeed
	GenesisBlock             *block.Block
	PowLimitBits             uint32
	CoinbaseMaturity         uint16
	SubsidyReductionInterval int32
	RetargetAdjustmentFactor int64
	ReduceMinDifficulty      bool
	MinDiffReductionTime     time.Duration
	GenerateSupported        bool
	Checkpoints              []*Checkpoint
	MineBlocksOnDemands      bool

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

	PruneAfterHeight         int
	chainTxData              ChainTxData
	MiningRequiresPeers      bool
	DefaultConsistencyChecks bool
	fMineBlocksOnDemand      bool
}

func (param *BitcoinParams) TxData() *ChainTxData {
	return &param.chainTxData
}

var MainNetParams = BitcoinParams{
	Param: consensus.Param{
		GenesisHash:            &GenesisBlockHash,
		SubsidyHalvingInterval: 210000,
		BIP34Height:            227931,
		//little endian
		BIP34Hash: *util.HashFromString("000000000000024b89b42a942fe0d9fea3bb44ab7bd1b19115dd6a759c0808b8"),
		// 000000000000000004c2b624ed5d7756c508d90fd0da2c7c679febfa6c4735f0
		BIP65Height: 388381,
		// 00000000000000000379eaa19dce8c9b722d46ae6a57c2f1a988119488b50931
		BIP66Height: 363725,
		// 000000000000000004a1b34462cb8aeebd5799177f7a29cf28f2d1961716b5b5
		CSVHeight:                     419328,
		PowLimit:                      mainPowLimit,
		TargetTimespan:                60 * 60 * 24 * 14,
		TargetTimePerBlock:            60 * 10,
		FPowAllowMinDifficultyBlocks:  false,
		FPowNoRetargeting:             false,
		RuleChangeActivationThreshold: 1916,
		MinerConfirmationWindow:       2016,
		Deployments: [consensus.MaxVersionBitsDeployments]consensus.BIP9Deployment{
			consensus.DeploymentTestDummy: {Bit: 28, StartTime: 1199145601, Timeout: 1230767999},
			consensus.DeploymentCSV:       {Bit: 0, StartTime: 1462060800, Timeout: 1493596800},
		},

		// The best chain should have at least this much work.
		MinimumChainWork: *util.HashFromString("000000000000000000000000000000000000000000a0f3064330647e2f6c4828"),

		// By default assume that the signatures in ancestors of this block are
		// valid.
		DefaultAssumeValid: *util.HashFromString("000000000000000000e45ad2fbcc5ff3e85f0868dd8f00ad4e92dffabe28f8d2"),

		UAHFHeight: 478558,

		// November 13, 2017 hard fork
		DAAHeight: 504031,

		// May 15, 2018 hard fork
		MonolithActivationTime: 1526400000,

		// Nov 15, 2018 hard fork
		MagneticAnomalyActivationTime: 1542300000,

		// Wed, 15 May 2019 12:00:00 UTC hard fork
		GreatWallActivationTime: 1557921600,
	},

	Name:        "main",
	BitcoinNet:  wire.MainNet,
	DefaultPort: "8333",
	DNSSeeds: []DNSSeed{
		{Host: "seed.bitcoinabc.org", HasFiltering: true},                  // Pieter Wuille
		{Host: "seed-abc.bitcoinforks.org", HasFiltering: true},            // Matt Corallo
		{Host: "btccash-seeder.bitcoinunlimited.info", HasFiltering: true}, // Chris Decker
		{Host: "seed.bitprim.org", HasFiltering: true},
		{Host: "seed.deadalnix.me", HasFiltering: true},
		{Host: "seeder.criptolayer.net", HasFiltering: false},
	},
	GenesisBlock: GenesisBlock,

	PowLimitBits:             GenesisBlock.Header.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,

	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0,
	GenerateSupported:        false,
	Checkpoints: []*Checkpoint{
		{11111, util.HashFromString("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
		{33333, util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")},
		{74000, util.HashFromString("0000000000573993a3c9e41ce34471c079dcf5f52a0e824a81e7f953b8661a20")},
		{105000, util.HashFromString("00000000000291ce28027faea320c8d2b054b2e0fe44a773f3eefb151d6bdc97")},
		{134444, util.HashFromString("00000000000005b12ffd4cd315cd34ffd4a594f430ac814c91184a0d42d2b0fe")},
		{168000, util.HashFromString("000000000000099e61ea72015e79632f216fe6cb33d7899acb35b75c8303b763")},
		{193000, util.HashFromString("000000000000059f452a5f7340de6682a977387c17010ff6e6c3bd83ca8b1317")},
		{210000, util.HashFromString("000000000000048b95347e83192f69cf0366076336c639f9b7228e9ba171342e")},
		{216116, util.HashFromString("00000000000001b4f4b433e81ee46494af945cf96014816a4e2370f11b23df4e")},
		{225430, util.HashFromString("00000000000001c108384350f74090433e7fcf79a606b8e797f065b130575932")},
		{250000, util.HashFromString("000000000000003887df1f29024b06fc2200b55f8af8f35453d7be294df2d214")},
		{267300, util.HashFromString("000000000000000a83fbd660e918f218bf37edd92b748ad940483c7c116179ac")},
		{279000, util.HashFromString("0000000000000001ae8c72a0b0c301f67e3afca10e819efa9041e458e9bd7e40")},
		{295000, util.HashFromString("00000000000000004d9b4ef50f0f9d686fd69db2e03af35a100370c64632a983")},
		{300255, util.HashFromString("0000000000000000162804527c6e9b9f0563a280525f9d08c12041def0a0f3b2")},
		{319400, util.HashFromString("000000000000000021c6052e9becade189495d1c539aa37c58917305fd15f13b")},
		{343185, util.HashFromString("0000000000000000072b8bf361d01a6ba7d445dd024203fafc78768ed4368554")},
		{352940, util.HashFromString("000000000000000010755df42dba556bb72be6a32f3ce0b6941ce4430152c9ff")},
		{382320, util.HashFromString("00000000000000000a8dc6ed5b133d0eb2fd6af56203e4159789b092defd8ab2")},
		// UAHF fork block.
		{478558, util.HashFromString("0000000000000000011865af4122fe3b144e2cbeea86142e8ff2fb4107352d43")},
		// Nov, 13 DAA activation block.
		{504031, util.HashFromString("0000000000000000011ebf65b60d0a3de80b8175be709d653b4c1a1beeb6ab9c")},
		// Monolith activation.
		{530359, util.HashFromString("0000000000000000011ada8bd08f46074f44a8f155396f43e38acf9501c49103")},
		//Magnetic anomaly activation.
		//{556767, util.HashFromString("0000000000000000004626ff6e3b936941d341c5932ece4357eeccac44e6d56c")},
	},
	MineBlocksOnDemands: false,
	// Enforce current block version once majority of the network has
	// upgraded.
	// 75% (750 / 1000)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 95% (950 / 1000)
	BlockEnforceNumRequired: 750,
	BlockRejectNumRequired:  950,
	BlockUpgradeNumToCheck:  1000,

	RequireStandard:     true,
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
}

var TestNetParams = BitcoinParams{
	Param: consensus.Param{
		SubsidyHalvingInterval: 210000,
		BIP34Height:            21111,
		BIP34Hash:              *util.HashFromString("0000000023b3a96d3484e5abb3755c413e7d41500f8e2a5c3f0dd01299cd8ef8"),
		BIP65Height:            581885,
		BIP66Height:            330776,
		CSVHeight:              770112,
		//AntiReplayOpReturnSunsetHeight: 1250000,
		//AntiReplayOpReturnCommitment:   []byte("Bitcoin: A Peer-to-Peer Electronic Cash System"),
		PowLimit:                      testNetPowLimit,
		TargetTimespan:                60 * 60 * 24 * 14,
		TargetTimePerBlock:            60 * 10,
		FPowAllowMinDifficultyBlocks:  true,
		FPowNoRetargeting:             false,
		RuleChangeActivationThreshold: 1512,
		MinerConfirmationWindow:       2016,
		Deployments: [consensus.MaxVersionBitsDeployments]consensus.BIP9Deployment{
			consensus.DeploymentTestDummy: {
				Bit:       28,
				StartTime: 1199145601,
				Timeout:   1230767999,
			},
			consensus.DeploymentCSV: {
				Bit:       0,
				StartTime: 1456790400,
				Timeout:   1493596800,
			},
		},
		MinimumChainWork:   *util.HashFromString("00000000000000000000000000000000000000000000002a650f6ff7649485da"),
		DefaultAssumeValid: *util.HashFromString("0000000000327972b8470c11755adf8f4319796bafae01f5a6650490b98a17db"),
		UAHFHeight:         1155875,
		DAAHeight:          1188697,
		// May 15, 2018 hard fork
		MonolithActivationTime: 1526400000,

		// Nov 15, 2018 hard fork
		MagneticAnomalyActivationTime: 1542300000,
		// Wed, 15 May 2019 12:00:00 UTC hard fork
		GreatWallActivationTime: 1557921600,
		//CashHardForkActivationTime: 1510600000,
		GenesisHash: &TestNetGenesisHash,
		//CashaddrPrefix: "xbctest",
	},

	Name:        "test",
	BitcoinNet:  wire.TestNet3,
	DiskMagic:   wire.TestDiskMagic,
	DefaultPort: "18333",
	DNSSeeds: []DNSSeed{
		{Host: "testnet-seed.bitcoinabc.org", HasFiltering: true},
		{Host: "testnet-seed-abc.bitcoinforks.org", HasFiltering: true},
		{Host: "testnet-seed.deadalnix.me", HasFiltering: true},
		{Host: "testnet-seed.bitprim.org", HasFiltering: true},
		{Host: "testnet-seeder.criptolayer.net", HasFiltering: true},
	},
	GenesisBlock:             TestNetGenesisBlock,
	PowLimitBits:             GenesisBlock.Header.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 210000,
	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20,
	GenerateSupported:        false,
	Checkpoints: []*Checkpoint{
		{546, util.HashFromString("000000002a936ca763904c3c35fce2f3556c559c0214345d31b1bcebf76acb70")},
		// UAHF fork block.
		{1155875, util.HashFromString("00000000f17c850672894b9a75b63a1e72830bbd5f4c8889b5c1a80e7faef138")},
		// Nov, 13. DAA activation block.
		{1188697, util.HashFromString("0000000000170ed0918077bde7b4d36cc4c91be69fa09211f748240dabe047fb")},
	},
	MineBlocksOnDemands: false,
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
	HDCoinType:          1,
	MiningRequiresPeers: true,

	chainTxData: ChainTxData{time.Unix(1483546230, 0), 12834668, 0.15},
}

var RegressionNetParams = BitcoinParams{
	Param: consensus.Param{
		GenesisHash:                   &RegTestGenesisHash,
		SubsidyHalvingInterval:        150,
		BIP34Height:                   100000000,
		BIP34Hash:                     util.Hash{},
		BIP65Height:                   1351,
		BIP66Height:                   1251,
		CSVHeight:                     576,
		PowLimit:                      regressingPowLimit,
		TargetTimespan:                60 * 60 * 24 * 14,
		TargetTimePerBlock:            60 * 10,
		FPowAllowMinDifficultyBlocks:  true,
		FPowNoRetargeting:             true,
		RuleChangeActivationThreshold: 108,
		MinerConfirmationWindow:       144,
		Deployments: [consensus.MaxVersionBitsDeployments]consensus.BIP9Deployment{
			consensus.DeploymentTestDummy: {
				Bit:       28,
				StartTime: 0,
				Timeout:   999999999999,
			},
			consensus.DeploymentCSV: {
				Bit:       0,
				StartTime: 0,
				Timeout:   999999999999,
			},
		},
		MinimumChainWork:   *util.HashFromString("00"),
		DefaultAssumeValid: *util.HashFromString("00"),
		UAHFHeight:         0,
		DAAHeight:          0,

		// May 15, 2018 hard fork.
		MonolithActivationTime: 1526400000,

		// Nov 15, 2018 hard fork
		MagneticAnomalyActivationTime: 1542300000,

		// Wed, 15 May 2019 12:00:00 UTC hard fork
		GreatWallActivationTime: 1557921600,
	},

	Name:         "regtest",
	BitcoinNet:   wire.RegTestNet,
	DefaultPort:  "18444",
	DNSSeeds:     []DNSSeed{},
	GenesisBlock: RegTestGenesisBlock,

	PowLimitBits:             RegTestGenesisBlock.Header.Bits,
	CoinbaseMaturity:         100,
	SubsidyReductionInterval: 150,

	RetargetAdjustmentFactor: 4,
	ReduceMinDifficulty:      true,
	MinDiffReductionTime:     time.Minute * 20,
	GenerateSupported:        true,
	Checkpoints:              nil,
	MineBlocksOnDemands:      true,
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

var (
	RegisteredNets          = make(map[wire.BitcoinNet]struct{})
	PubKeyHashAddressIDs    = make(map[byte]struct{})
	ScriptHashAddressIDs    = make(map[byte]struct{})
	HDPrivateToPublicKeyIDs = make(map[[4]byte][]byte)
)

func init() {
	mustRegister(&MainNetParams)
	mustRegister(&TestNetParams)
	mustRegister(&RegressionNetParams)
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
func mustRegister(bp *BitcoinParams) {
	err := Register(bp)
	if err != nil {
		panic("failed to register network :" + err.Error())
	}
}

//IsUAHFEnabled Check is UAHF has activated.
func IsUAHFEnabled(height int32) bool {
	return height >= ActiveNetParams.UAHFHeight
}

func IsMonolithEnabled(medianTimePast int64) bool {
	time := ActiveNetParams.MonolithActivationTime
	if conf.Args.MonolithActivationTime > 0 {
		time = conf.Args.MonolithActivationTime
	}

	return medianTimePast >= time
}

func IsDAAEnabled(height int32) bool {
	return height >= ActiveNetParams.DAAHeight
}

func IsMagneticAnomalyEnabled(mediaTimePast int64) bool {
	activeTime := ActiveNetParams.MagneticAnomalyActivationTime
	if conf.Args.MagneticAnomalyTime > 0 {
		activeTime = conf.Args.MagneticAnomalyTime
	}
	return mediaTimePast >= activeTime
}

func IsReplayProtectionEnabled(medianTimePast int64) bool {
	time := ActiveNetParams.GreatWallActivationTime
	if conf.Args.ReplayProtectionActivationTime > 0 {
		time = conf.Args.ReplayProtectionActivationTime
	}

	return medianTimePast >= time
}

func SetTestNetParams() {
	ActiveNetParams = &TestNetParams
	setActiveNetAddressParams()
}

func SetRegTestParams() {
	ActiveNetParams = &RegressionNetParams
	setActiveNetAddressParams()
}

func setActiveNetAddressParams() {
	script.InitAddressParam(&script.AddressParam{
		PubKeyHashAddressVer: ActiveNetParams.PubKeyHashAddressID,
		ScriptHashAddressVer: ActiveNetParams.ScriptHashAddressID,
	})
	crypto.InitPrivateKeyVersion(ActiveNetParams.PrivatekeyID)
}

func GetBlockSubsidy(height int32, params *BitcoinParams) amount.Amount {
	halvings := height / params.SubsidyReductionInterval
	// Force block reward to zero when right shift is undefined.
	if halvings >= 64 {
		return 0
	}

	nSubsidy := amount.Amount(50 * util.COIN)
	// Subsidy is cut in half every 210,000 blocks which will occur
	// approximately every 4 years.
	return amount.Amount(uint(nSubsidy) >> uint(halvings))
}
