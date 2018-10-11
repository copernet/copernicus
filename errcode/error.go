package errcode

import (
	"fmt"
)

const (
	MempoolErrorBase = iota * 1000
	ScriptErrorBase
	TxErrorBase
	ChainErrorBase
	RPCErrorBase
	DiskErrorBase
)

type ProjectError struct {
	Module string
	Code   int
	Desc   string

	ErrorCode fmt.Stringer
}

func (e ProjectError) Error() string {
	return fmt.Sprintf("module: %s, global errcode: %v,  desc: %s", e.Module, e.Code, e.Desc)
}

func getCode(errCode fmt.Stringer) (int, string) {
	code := 0
	module := ""

	switch t := errCode.(type) {
	case RPCErr:
		code = int(t)
		module = "rpc"
	case MemPoolErr:
		code = int(t)
		module = "mempool"
	case ChainErr:
		code = int(t)
		module = "chain"
	case DiskErr:
		code = int(t)
		module = "disk"
	case ScriptErr:
		code = int(t)
		module = "script"
	case TxErr:
		code = int(t)
		module = "transaction"
	case RejectCode:
		code = int(t)
		module = "tx_validation"
	default:
	}

	return code, module
}

func IsErrorCode(err error, errCode fmt.Stringer) bool {
	e, ok := err.(ProjectError)
	code, _ := getCode(errCode)
	return ok && code == e.Code
}

func New(errCode fmt.Stringer) error {
	return NewError(errCode, errCode.String())
}

func NewError(errCode fmt.Stringer, desc string) error {
	code, module := getCode(errCode)

	return ProjectError{
		Module:    module,
		Code:      code,
		Desc:      desc,
		ErrorCode: errCode,
	}
}

func HasRejectCode(err error) (RejectCode, bool) {
	pe, ok := err.(ProjectError)
	if ok && pe.ErrorCode != nil {
		switch t := pe.ErrorCode.(type) {
		case RejectCode:
			return t, true
		}
	}

	return 0, false
}
