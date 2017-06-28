package connect

import "github.com/pkg/errors"

const (
	TOR_SUCCEEDED             = 0x00
	TOR_GENERAL_ERROR         = 0x01
	TOR_NOT_ALLOWED           = 0X02
	TOR_NET_UNREACHABLE       = 0X03
	TOR_HOST_UNREACHABLE      = 0x04
	TOR_CONNECTION_REFUSED    = 0x05
	TOR_TTL_EXPIRED           = 0x06
	TOR_CMD_NOT_SUPPORTED     = 0x07
	TOR_ADDRESS_NOT_SUPPORTED = 0x08
)

var torStatusErrors = map[byte]error{
	TOR_SUCCEEDED:             errors.New("tor succeeded"),
	TOR_GENERAL_ERROR:         errors.New("tor general error"),
	TOR_NOT_ALLOWED:           errors.New("tor not allowed"),
	TOR_NET_UNREACHABLE:       errors.New("tor network is unreachable"),
	TOR_HOST_UNREACHABLE:      errors.New("tor host is unreachable"),
	TOR_CONNECTION_REFUSED:    errors.New("tor connection refused"),
	TOR_TTL_EXPIRED:           errors.New("tor TTL expired"),
	TOR_CMD_NOT_SUPPORTED:     errors.New("tor command not supported"),
	TOR_ADDRESS_NOT_SUPPORTED: errors.New("tor address type not supported"),
}
