package conf

import (
	"github.com/btcboost/copernicus/util"
)

const (
	// DefaultBytesPerSigOP default for -bytesPerSigOP
	DefaultBytesPerSigOP uint = 20

	// DustRelayTxFee min feeRate for defining dust. Historically this has been the same as the
	// minRelayTxFee, however changing the dust limit changes which transactions are
	// standard and should be done with care and ideally rarely. It makes sense to
	// only increase the dust limit after prior releases were already not creating
	// outputs below the new threshold.
	DustRelayTxFee uint = 1000

	// DefaultIncrementalRelayFee default for -incrementAlrelayFee, which sets the minimum feeRate increase
	// for mempool limiting or BIP 125 replacement
	DefaultIncrementalRelayFee uint = 1000

	// DefaultPermitBareMultiSig Default for -permitbaremultisig
	DefaultPermitBareMultiSig      = true
	DefaultCheckPointsEnabled      = true
	DefaultTxIndex                 = false
	DefaultBanScoreThreshold  uint = 100

	DefaultAcceptDataCarrier = true

	// MaxOpReturnRelay bytes (+1 for OP_RETURN, +2 for the pushdata opcodes)
	MaxOpReturnRelay uint = 83
)

type GlobalValue struct {
	isBareMultiSigStd   bool
	incrementalRelayFee util.FeeRate
	dustRelayFee        util.FeeRate
	bytesPerSigOp       uint
	maxDataCarrierBytes uint
	acceptDataCarrier   bool
}

var GlobalValueInstance GlobalValue

func init() {
	GlobalValueInstance.isBareMultiSigStd = DefaultPermitBareMultiSig
	GlobalValueInstance.incrementalRelayFee = util.FeeRate{SataoshisPerK: int64(DefaultIncrementalRelayFee)}
	GlobalValueInstance.bytesPerSigOp = DefaultBytesPerSigOP
	GlobalValueInstance.dustRelayFee = util.FeeRate{SataoshisPerK: int64(DustRelayTxFee)}
	GlobalValueInstance.acceptDataCarrier = DefaultAcceptDataCarrier
	GlobalValueInstance.maxDataCarrierBytes = MaxOpReturnRelay
}

func (g *GlobalValue) GetAcceptDataCarrier() bool {
	return g.acceptDataCarrier
}

func (g *GlobalValue) SetAcceptDataCarrier(flag bool) {
	g.acceptDataCarrier = flag
}

func (g *GlobalValue) GetMaxDataCarrierBytes() uint {
	return g.maxDataCarrierBytes
}

func (g *GlobalValue) SetMaxDataCarrierBytes(flag uint) {
	g.maxDataCarrierBytes = flag
}

func (g *GlobalValue) SetIsBareMultiSigStd(flag bool) {
	g.isBareMultiSigStd = flag
}

func (g *GlobalValue) GetIsBareMultiSigStd() bool {
	return g.isBareMultiSigStd
}

func (g *GlobalValue) SetIncrementalRelayFee(fee util.FeeRate) {
	g.incrementalRelayFee = fee
}

func (g *GlobalValue) GetIncrementalRelayFee() util.FeeRate {
	return g.incrementalRelayFee
}

func (g *GlobalValue) SetDustRelayFee(fee util.FeeRate) {
	g.dustRelayFee = fee
}

func (g *GlobalValue) GetDustRelayFee() util.FeeRate {
	return g.dustRelayFee
}

func (g *GlobalValue) SetBytesPerSigOp(flag uint) {
	g.bytesPerSigOp = flag
}

func (g *GlobalValue) GetBytesPerSigOp() uint {
	return g.bytesPerSigOp
}
