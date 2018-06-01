package consensus

import (
	"math/big"
	"time"

	"github.com/btcboost/copernicus/util"
)

type DeploymentPos int

const (
	DeploymentTestDummy DeploymentPos = iota
	// DeploymentCSV deployment of BIP68, BIP112, and BIP113.
	DeploymentCSV
	// MaxVersionBitsDeployments NOTE: Also add new deployments to VersionBitsDeploymentInfo in
	// versionbits.cpp
	MaxVersionBitsDeployments
)

type BIP9Deployment struct {
	/** Bit position to select the particular bit in nVersion. */
	Bit int
	/** Start MedianTime for version bits miner confirmation. Can be a date in
	 * the past */
	StartTime int64
	/** Timeout/expiry MedianTime for the deployment attempt. */
	Timeout int64
}

type Param struct {
	GenesisHash            *util.Hash
	SubsidyHalvingInterval int
	// Block height and hash at which BIP34 becomes active
	BIP34Height int32
	BIP34Hash   util.Hash
	//  Block height at which BIP65 becomes active
	BIP65Height int32
	//  Block height at which BIP66 becomes active
	BIP66Height int32
	//  Block height at which UAHF kicks in
	UAHFHeight int32

	// Block height at which OP_RETURN replay protection stops
	AntiReplayOpReturnSunsetHeight int32
	AntiReplayOpReturnCommitment   []byte

	// Minimum blocks including miner confirmation of the total of 2016 blocks
	// in a retargeting period, (nPowTargetTimespan / nPowTargetSpacing) which
	// is also used for BIP9 deployments.
	// Examples: 1916 for 95%, 1512 for testchains.
	RuleChangeActivationThreshold uint32

	MinerConfirmationWindow uint32

	Deployments [MaxVersionBitsDeployments]BIP9Deployment
	// Proof of work parameters
	PowLimit                     *big.Int
	FPowAllowMinDifficultyBlocks bool
	FPowNoRetargeting            bool
	TargetTimePerBlock           time.Duration
	TargetTimespan               time.Duration

	// The best chain should have at least this much work.
	MinimumChainWork util.Hash

	// By default assume that the signatures in ancestors of this block are valid.
	DefaultAssumeValid util.Hash

	//  Activation time at which the cash HF kicks in.
	CashHardForkActivationTime int64

	CashaddrPrefix 		string
}

func (pm *Param) DifficultyAdjustmentInterval() int64 {
	return int64(pm.TargetTimespan / pm.TargetTimePerBlock)
}
