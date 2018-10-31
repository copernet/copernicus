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
	return fmt.Sprintf("module: %s, errcode: %v: %s", e.Module, e.Code, e.Desc)
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

func MakeError(code RejectCode, format string, innerErr error) error {
	return NewError(code, fmt.Sprintf(format, shortDesc(innerErr)))
}

// IsRejectCode BIP61 reject code; never send internal reject codes over P2P.
func IsRejectCode(err error) (RejectCode, string, bool) {
	e, ok := err.(ProjectError)
	if ok && e.ErrorCode != nil {
		switch t := e.ErrorCode.(type) {
		case RejectCode:
			return t, e.Desc, true
		}
	}

	return 0, "", false
}

func shortDesc(err error) string {
	e, ok := err.(ProjectError)
	if ok && e.ErrorCode != nil {
		return e.ErrorCode.String()
	}

	return e.Error()
}
