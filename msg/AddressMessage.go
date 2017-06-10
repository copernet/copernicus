package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"

)

const (
	MAX_ADDRESSES_COUNT = 1024
)

type AddressMesage struct {
	Message
	AddressList []*PeerAddress
}

func (addressMsg *AddressMesage) AddPeerAddress(peerAddress *PeerAddress) error {
	if len(addressMsg.AddressList) > MAX_ADDRESSES_COUNT {
		str := fmt.Sprintf("has too many addresses in message ,count is %v ", MAX_ADDRESSES_COUNT)
		return errors.New(str)
	}
	addressMsg.AddressList = append(addressMsg.AddressList, peerAddress)
	return nil
}

func (addressMsg *AddressMesage) AddPeerAddresses(peerAddresses ...*PeerAddress) (err error) {
	for _, peerAddress := range peerAddresses {
		err = addressMsg.AddPeerAddress(peerAddress)
		if err != nil {
			return err
		}
	}
	return nil
}

func (addressMsg *AddressMesage) ClearAddresses() {
	addressMsg.AddressList = []*PeerAddress{}
}

func (msg*AddressMesage) BitcoinParse(reader io.Reader, size uint32) {
	count,err:=store.


}
