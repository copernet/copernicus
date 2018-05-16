package errcode

import "fmt"

type chainErr int

const (
	HeaderNotValid chainErr = 3000 + iota

)

var chainDesc = map[chainErr] string {
	HeaderNotValid:"the block header status is not valid",
}

func (cErr chainErr) String() string {
	if s, ok := chainDesc[cErr]; ok {
		return s
	}
	return fmt.Sprintf("Unknown code (%d)", cErr)
}