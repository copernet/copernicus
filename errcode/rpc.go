package errcode

type RPCErr int

const RPCBase RPCErr = 2000

const (
	ModelValid = RPCBase + iota
	ModelInvalid
	ModelError
)

var rpcDesc = map[RPCErr]string{
	ModelValid:   "valid",
	ModelInvalid: "invalid",
	ModelError:   "error",
}

func (re RPCErr) String() string {
	msg, ok := rpcDesc[re]
	if ok {
		return msg
	}

	return "Unknown error code!"
}
