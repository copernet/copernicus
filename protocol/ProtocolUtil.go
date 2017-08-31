package protocol

import (
	"fmt"

	"github.com/pkg/errors"
)

func ValidateUserAgent(useAgent string) error {
	if len(useAgent) > MaxUserAgentLen {
		str := fmt.Sprintf("userAgent len : %v,max : %v", len(useAgent), MaxUserAgentLen)
		return errors.New(str)

	}
	return nil

}
