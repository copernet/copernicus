package protocol

import (
	"fmt"
	"github.com/pkg/errors"
)

func ValidateUserAgent(useAgent string) error {
	if len(useAgent) > MAX_USER_AGENT_LEN {
		str := fmt.Sprintf("userAgent len : %v,max : %v", len(useAgent), MAX_USER_AGENT_LEN)
		return errors.New(str)

	}
	return nil

}
