package errcode

import (
	"fmt"
)

const (
	MempoolErrorBase = iota * 1000
	ScriptErrorBase
	TxErrorBase
	TxOutErrorBase
	ChainErrorBase
	BlockErrorBase
	BlockIndexErrorBase
	CoinErrorBase
	MessageErrorBase
	RpcErrorBase
	NetErrorBase
	PeerErrorBase
	ServiceErrorBase
	PersistErrorBase
	CryptoErrorBase
	ConsensusErrorBase
	DiskErrorBase
)

const errDescFmt string = "module: [%s], inner err desc: [%s]"

type ProjectError struct {
	Module string
	Code   int
	Desc   string
}

func (e ProjectError) Error() string {
	return fmt.Sprintf("module: %s, global errcode: %v,  errdesc: %s", e.Module, e.Code, e.Desc)
}

func getCodeAndName(errCode fmt.Stringer) (int, string) {
	code := 0
	name := ""

	switch t := errCode.(type) {
	case RPCErr:
		code = int(t)
		name = "rpc"
	case MemPoolErr:
		code = int(t)
		name = "mempool"
	case ChainErr:
		code = int(t)
		name = "chain"
	case DiskErr:
		code = int(t)
		name = "disk"
	case ScriptErr:
		code = int(t)
		name = "script"
	case TxErr:
		code = int(t)
		name = "transaction"
	case TxOutErr:
		code = int(t)
		name = "transaction"
	default:
	}

	return code, name
}

func IsErrorCode(err error, errCode fmt.Stringer) bool {
	e, ok := err.(ProjectError)
	icode, _ := getCodeAndName(errCode)
	return ok && icode == e.Code
}

func New(errCode fmt.Stringer) error {
	code, name := getCodeAndName(errCode)

	return ProjectError{
		Module: name,
		Code:   code,
		Desc:   errCode.String(),
	}
}
