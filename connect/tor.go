package connect

import "github.com/pkg/errors"

const (
	TorSucceeded           = 0x00
	TorGeneralError        = 0x01
	TorNotAllowed          = 0X02
	TorNetUnreachable      = 0X03
	TorHostUnreachable     = 0x04
	TorConnectionRefused   = 0x05
	TorTTLExpired          = 0x06
	TorCMDNotSupported     = 0x07
	TorAddressNotSupported = 0x08
)

var torStatusErrors = map[byte]error{
	TorSucceeded:           errors.New("tor succeeded"),
	TorGeneralError:        errors.New("tor general error"),
	TorNotAllowed:          errors.New("tor not allowed"),
	TorNetUnreachable:      errors.New("tor network is unreachable"),
	TorHostUnreachable:     errors.New("tor host is unreachable"),
	TorConnectionRefused:   errors.New("tor connection refused"),
	TorTTLExpired:          errors.New("tor TTL expired"),
	TorCMDNotSupported:     errors.New("tor command not supported"),
	TorAddressNotSupported: errors.New("tor address type not supported"),
}
