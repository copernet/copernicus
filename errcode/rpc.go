package errcode

type RPCErr int

const (
	ModelValid RPCErr = RPCErrorBase + iota
	ModelInvalid
	ModelError
	ErrorNotExistInRPCMap
)

var rpcDesc = map[RPCErr]string{
	ModelValid:   "Valid",
	ModelInvalid: "Invalid",
	ModelError:   "Error",
}

func (re RPCErr) String() string {
	msg, ok := rpcDesc[re]
	if ok {
		return msg
	}

	return "Unknown error code!"
}
