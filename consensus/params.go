package consensus

import "github.com/btcboost/copernicus/utils"

type DeploymentPos int

const (
	DEPLOYMENT_TESTDUMMY DeploymentPos = iota
	// DEPLOYMENT_CSV Deployment of BIP68, BIP112, and BIP113.
	DEPLOYMENT_CSV
	// MAX_VERSION_BITS_DEPLOYMENTS NOTE: Also add new deployments to VersionBitsDeploymentInfo in
	// versionbits.cpp
	MAX_VERSION_BITS_DEPLOYMENTS
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

type Params struct {
	hashGenesisBlock       utils.Hash
	SubsidyHalvingInterval int
	/** Block height and hash at which BIP34 becomes active */
	BIP34Height int
	BIP34Hash   utils.Hash
	/** Block height at which BIP65 becomes active */
	BIP65Height int
	/** Block height at which BIP66 becomes active */
	BIP66Height int
	/** Block height at which UAHF kicks in */
	uahfHeight int
	/** Block height at which OP_RETURN replay protection stops */
	antiReplayOpReturnSunsetHeight int
	/** Committed OP_RETURN value for replay protection */
	antiReplayOpReturnCommitment []uint8
	/**
	 * Minimum blocks including miner confirmation of the total of 2016 blocks
	 * in a retargeting period, (nPowTargetTimespan / nPowTargetSpacing) which
	 * is also used for BIP9 deployments.
	 * Examples: 1916 for 95%, 1512 for testchains.
	 */
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow       uint32
	Deployments                   [MAX_VERSION_BITS_DEPLOYMENTS]BIP9Deployment
	/** Proof of work parameters */
	powLimit                    utils.Hash
	PowAllowMinDifficultyBlocks bool
	PowNoRetargeting            bool
	PowTargetSpacing            int64
	PowTargetTimespan           int64

	MinimumChainWork   utils.Hash
	defaultAssumeValid utils.Hash

	/** Activation time at which the cash HF kicks in. */
	cashHardForkActivationTime int64
}

func (params *Params) DifficultyAdjustmentInterval() int64 {
	return params.PowTargetTimespan / params.PowTargetSpacing
}
