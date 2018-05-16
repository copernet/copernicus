package errcode

import "fmt"

type ChainErr int

const (
	ErrorBlockHeaderNoValid ChainErr = ChainErrorBase + iota
)

var ChainErrString = map[ChainErr]string {
	ErrorBlockHeaderNoValid: "The block header is not valid",
}

func (chainerr ChainErr) String() string {
	if s, ok := ChainErrString[chainerr]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)",chainerr)
}
