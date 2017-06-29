package msg

import (
	"fmt"
	"github.com/btccom/copernicus/network"
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/utils"
	"github.com/pkg/errors"
	"io"
)

const (
	MaxAddressesCount = 1024
	MaxVarIntPayload  = 9
	MaxUserAgentLen   = 256
)

type AddressMessage struct {
	Message
	AddressList []*network.PeerAddress
}

func (addressMessage *AddressMessage) AddPeerAddress(peerAddress *network.PeerAddress) error {
	if len(addressMessage.AddressList) > MaxAddressesCount {
		str := fmt.Sprintf("has too many addresses in message ,count is %v ", MaxAddressesCount)
		return errors.New(str)
	}
	addressMessage.AddressList = append(addressMessage.AddressList, peerAddress)
	return nil
}

func (addressMessage *AddressMessage) AddPeerAddresses(peerAddresses ...*network.PeerAddress) (err error) {
	for _, peerAddress := range peerAddresses {
		err = addressMessage.AddPeerAddress(peerAddress)
		if err != nil {
			return err
		}
	}
	return nil
}

func (addressMessage *AddressMessage) ClearAddresses() {
	addressMessage.AddressList = []*network.PeerAddress{}
}

func (addressMessage *AddressMessage) BitcoinParse(reader io.Reader, size uint32) error {
	count, err := utils.ReadVarInt(reader, size)
	if err != nil {
		return err
	}
	if count > MaxAddressesCount {
		str := fmt.Sprintf("too many addresses for message count %v ,max %v", count, MaxAddressesCount)
		return errors.New(str)
	}
	addrList := make([]*network.PeerAddress, count)
	addressMessage.AddressList = make([]*network.PeerAddress, 0, count)
	for i := uint64(0); i < count; i++ {
		peerAddress := addrList[i]
		err := network.ReadPeerAddress(reader, size, peerAddress, true)
		if err != nil {
			return err
		}
		addressMessage.AddPeerAddress(peerAddress)
	}
	return nil

}

func (addressMessage *AddressMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	count := len(addressMessage.AddressList)
	if size < protocol.MultipleAddressVersion && count > 1 {
		str := fmt.Sprintf("too many address for message of protocol version %v count %v ", size, count)
		return errors.New(str)
	}
	if count > MaxAddressesCount {
		str := fmt.Sprintf("too many addresses for message count %v,max %v", count, MaxAddressesCount)
		return errors.New(str)

	}
	err := utils.WriteVarInt(w, size, uint64(count))
	if err != nil {
		return err
	}
	for _, peerAddress := range addressMessage.AddressList {
		err := network.WritePeerAddress(w, size, peerAddress, true)
		if err != nil {
			return err
		}
	}
	return nil

}

func (addressMessage *AddressMessage) MaxPayloadLength(version uint32) uint32 {
	if version < protocol.MultipleAddressVersion {
		return MaxVarIntPayload + network.MaxPeerAddressPayload(version)
	}
	return MaxVarIntPayload + (MaxAddressesCount * network.MaxPeerAddressPayload(version))
}

func (addressMessage *AddressMessage) Command() string {
	return CommandAddress
}
