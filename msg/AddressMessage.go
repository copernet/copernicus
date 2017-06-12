package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"copernicus/utils"
	"copernicus/protocol"
)

const (
	MAX_ADDRESSES_COUNT = 1024
	MAX_VAR_INT_PAYLOAD = 9
	MAX_USERAGENT_LEN=256
)

type AddressMessage struct {
	Message
	AddressList []*PeerAddress
}

func (addressMsg *AddressMessage) AddPeerAddress(peerAddress *PeerAddress) error {
	if len(addressMsg.AddressList) > MAX_ADDRESSES_COUNT {
		str := fmt.Sprintf("has too many addresses in message ,count is %v ", MAX_ADDRESSES_COUNT)
		return errors.New(str)
	}
	addressMsg.AddressList = append(addressMsg.AddressList, peerAddress)
	return nil
}

func (addressMsg *AddressMessage) AddPeerAddresses(peerAddresses ...*PeerAddress) (err error) {
	for _, peerAddress := range peerAddresses {
		err = addressMsg.AddPeerAddress(peerAddress)
		if err != nil {
			return err
		}
	}
	return nil
}

func (addressMsg *AddressMessage) ClearAddresses() {
	addressMsg.AddressList = []*PeerAddress{}
}

func (msg*AddressMessage) BitcoinParse(reader io.Reader, size uint32) error {
	count, err := utils.ReadVarInt(reader, size)
	if err != nil {
		return err
	}
	if count > MAX_ADDRESSES_COUNT {
		str := fmt.Sprintf("too many addresses for message count %v ,max %v", count, MAX_ADDRESSES_COUNT)
		return errors.New(str)
	}
	addrList := make([]*PeerAddress, count)
	msg.AddressList = make([]*PeerAddress, 0, count)
	for i := uint64(0); i < count; i++ {
		peerAddress := addrList[i]
		err := ReadPeerAddress(reader, size, peerAddress, true)
		if err != nil {
			return err
		}
		msg.AddPeerAddress(peerAddress)
	}
	return nil

}

func (addressMessage*AddressMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	count := len(addressMessage.AddressList)
	if size < protocol.MULTILE_ADDRESS_VERSION && count > 1 {
		str := fmt.Sprintf("too many address for message of protocol version %v count %v ", size, count)
		return errors.New(str)
	}
	if count > MAX_ADDRESSES_COUNT {
		str := fmt.Sprintf("too many addresses for message count %v,max %v", count, MAX_ADDRESSES_COUNT)
		return errors.New(str)

	}
	err := utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}
	for _, peerAddress := range addressMessage.AddressList {
		err := WritePeerAddress(w, size, peerAddress, true)
		if err != nil {
			return err
		}
	}
	return nil

}

func (addressMesage *AddressMessage) MaxPayloadLength(version uint32) uint32 {
	if version < protocol.MULTILE_ADDRESS_VERSION {
		return MAX_VAR_INT_PAYLOAD + MaxPeerAddressPayload(version)
	}
	return MAX_VAR_INT_PAYLOAD + (MAX_ADDRESSES_COUNT * MaxPeerAddressPayload(version))
}
