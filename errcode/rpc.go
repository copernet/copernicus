package errcode

type RpcErr int

const RpcBase RpcErr = 2000

const (
	ModelValid RpcErr = RpcBase + iota
	ModelInvalid
	ModelError
)

var rpcDesc = map[RpcErr]string{
	ModelValid:   "valid",
	ModelInvalid: "invalid",
	ModelError:   "error",
}

func (re RpcErr) String() string {
	msg, ok := rpcDesc[re]
	if ok {
		return msg
	}

	return "Unknown error code!"
}
