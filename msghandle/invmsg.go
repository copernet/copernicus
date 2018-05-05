package msghandle

import (
	"github.com/btcboost/copernicus/utils"
	"fmt"
)

type InvType uint32

// getdata / inv message types.
// These numbers are defined by the protocol. When adding a new value, be sure
// to mention it in the respective BIP.
const (
	UnDefined InvType = iota
	MsgTx
	MsgBlock

	// The following can only occur in getdata. Invs always use TX or BLOCK.
	// Defined in BIP37
	MsgFilteredBlock
	// Defined in BIP152
	MsgCmpctBlock
	MsgExtTx = MsgTx | MsgExtFlag
	MsgExtBlock = MsgBlock | MsgExtFlag

	// getdata message type flags
	MsgExtFlag = 1 << 29
	MsgTypeMask = 0xffffffff >> 3
)

var ivStrings = map[InvType]string{
	MsgTx: "tx",
	MsgBlock: "block",
	MsgFilteredBlock : "merkleblock",
	MsgCmpctBlock : "cmpctblock",
}

//inv message data
type Inv struct {
	typeID InvType
	hash utils.Hash
}

func NewInv(typ InvType, hash *utils.Hash) *Inv {
	return &Inv{
		typeID:typ,
		hash:*hash,
	}
}

func (inv *Inv)GetCommand() string {
	cmd := ""
	if inv.typeID & MsgExtFlag != 0{
		cmd = "extblk-"
	}
	switch inv.GetKind(){
	case MsgTx:
		return cmd + ivStrings[MsgTx]
	case MsgBlock:
		 return cmd + ivStrings[MsgBlock]
	case MsgFilteredBlock:
		return cmd + ivStrings[MsgFilteredBlock]
	case MsgCmpctBlock:
		return cmd+ivStrings[MsgCmpctBlock]
	default:
		panic(fmt.Sprintf("CInv::GetCommand(): type=%d unknown type ", inv.typeID))
	}
}

func (inv *Inv)ToString() (cmd string) {
	defer func() {
		if err := recover(); err != nil{
			return
		}
	}
	cmd := inv.GetCommand()


}

func (inv *Inv)GetKind() InvType {
	return inv.typeID & MsgTypeMask
}

func (inv *Inv)IsTx() bool {
	k := inv.GetKind()
	return k == MsgTx
}

func (inv *Inv)IsSomeBlock() bool {
	k := inv.GetKind()
	return k == MsgBlock || k == MsgFilteredBlock || k == MsgCmpctBlock
}

func (inv *Inv)Com(b *Inv) bool {
	return inv.typeID < b.typeID ||
		(inv.typeID == b.typeID && inv.hash.Cmp(&b.hash) < 0)
}