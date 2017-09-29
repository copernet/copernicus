package conf

import (
	"github.com/btcboost/copernicus/utils"
)

const (
	/*DEFAULT_BYTES_PER_SIGOP Default for -bytespersigop */
	DEFAULT_BYTES_PER_SIGOP uint = 20

	/*DUST_RELAY_TX_FEE Min feerate for defining dust. Historically this has been the same as the
	 * minRelayTxFee, however changing the dust limit changes which transactions are
	 * standard and should be done with care and ideally rarely. It makes sense to
	 * only increase the dust limit after prior releases were already not creating
	 * outputs below the new threshold.
	 */
	DUST_RELAY_TX_FEE uint = 1000

	/*DEFAULT_INCREMENTAL_RELAY_FEE Default for -incrementalrelayfee, which sets the minimum feerate increase
	 * for mempool limiting or BIP 125 replacement **/
	DEFAULT_INCREMENTAL_RELAY_FEE uint = 1000

	/*DEFAULT_PERMIT_BAREMULTISIG Default for -permitbaremultisig */
	DEFAULT_PERMIT_BAREMULTISIG bool = true
	DEFAULT_CHECKPOINTS_ENABLED bool = true
	DEFAULT_TXINDEX             bool = false
	DEFAULT_BANSCORE_THRESHOLD  uint = 100

	DEFAULT_ACCEPT_DATACARRIER = true

	//MAX_OP_RETURN_RELAY bytes (+1 for OP_RETURN, +2 for the pushdata opcodes)
	MAX_OP_RETURN_RELAY uint = 83
)

type GlobalValue struct {
	isBareMultisigStd   bool
	incrementalRelayFee utils.FeeRate
	dustRelayFee        utils.FeeRate
	bytesPerSigOp       uint
	maxDatacarrierBytes uint
	acceptDatacarrier   bool
}

var GlobalValueInstance GlobalValue

func init() {
	GlobalValueInstance.isBareMultisigStd = DEFAULT_PERMIT_BAREMULTISIG
	GlobalValueInstance.incrementalRelayFee = utils.FeeRate{SataoshisPerK: int64(DEFAULT_INCREMENTAL_RELAY_FEE)}
	GlobalValueInstance.bytesPerSigOp = DEFAULT_BYTES_PER_SIGOP
	GlobalValueInstance.dustRelayFee = utils.FeeRate{SataoshisPerK: int64(DUST_RELAY_TX_FEE)}
	GlobalValueInstance.acceptDatacarrier = DEFAULT_ACCEPT_DATACARRIER
	GlobalValueInstance.maxDatacarrierBytes = MAX_OP_RETURN_RELAY
}

func (g *GlobalValue) GetAcceptDatacarrier() bool {
	return g.acceptDatacarrier
}

func (g *GlobalValue) SetAcceptDatacarrier(flag bool) {
	g.acceptDatacarrier = flag
}

func (g *GlobalValue) GetMaxDatacarrierBytes() uint {
	return g.maxDatacarrierBytes
}

func (g *GlobalValue) SetMaxDatacarrierBytes(flag uint) {
	g.maxDatacarrierBytes = flag
}

func (g *GlobalValue) SetIsBareMultisigStd(flag bool) {
	g.isBareMultisigStd = flag
}

func (g *GlobalValue) GetIsBareMultisigStd() bool {
	return g.isBareMultisigStd
}

func (g *GlobalValue) SetIncrementalRelayFee(fee utils.FeeRate) {
	g.incrementalRelayFee = fee
}

func (g *GlobalValue) GetIncrementalRelayFee() utils.FeeRate {
	return g.incrementalRelayFee
}

func (g *GlobalValue) SetDustRelayFee(fee utils.FeeRate) {
	g.dustRelayFee = fee
}

func (g *GlobalValue) GetDustRelayFee() utils.FeeRate {
	return g.dustRelayFee
}

func (g *GlobalValue) SetBytesPerSigOp(flag uint) {
	g.bytesPerSigOp = flag
}

func (g *GlobalValue) GetBytesPerSigOp() uint {
	return g.bytesPerSigOp
}
