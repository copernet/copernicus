package errcode

import "github.com/copernet/copernicus/rpc/btcjson"

func GetBip22Result(rerr error) (err error) {

	if rerr == nil {
		return NewError(ModelValid, "")
	}

	perr, ok := rerr.(ProjectError)
	if !ok {
		return NewError(ModelError, "Unknown Error")
	}

	switch perr.Code {

	case int(btcjson.ErrRPCDeserialization):
		err = NewError(ModelError, "Block decode failed")
	case int(ErrorBlockNotStartWithCoinBase):
		err = NewError(ModelError, "Block does not start with a coinbase")

	// model invalid
	case int(ErrorbadTxnsDuplicate):
		err = NewError(ModelInvalid, perr.Desc)

		// reject invalid
	case int(RejectInvalid):
		if perr.Desc == "" {
			perr.Desc = "rejected"
		}

		err = NewError(ModelInvalid, perr.Desc)

	default:

	}

	return err
}
