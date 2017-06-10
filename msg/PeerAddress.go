package msg

import (
	"time"
	"net"
	"io"
	"copernicus/protocol"
	"encoding/binary"
	"copernicus/utils"
)

type PeerAddress struct {
	Timestamp    time.Time
	ServicesFlag uint64
	IP           net.IP
	Port         uint16
}
type PeerAddressFunc func(remoteAddr *PeerAddress) *PeerAddress

func (na *PeerAddress) EqualService(serviceFlag uint64) bool {
	return na.ServicesFlag&serviceFlag == serviceFlag
}

func (na *PeerAddress) AddService(serviceFlag uint64) {
	na.ServicesFlag |= serviceFlag
}
func InitPeerAddressIPPort(serviceFlag uint64, ip net.IP, port uint16) *PeerAddress {
	return InitPeerAddress(time.Now(), serviceFlag, ip, port)
}

func InitPeerAddress(timestamp time.Time, serviceFlag uint64, ip net.IP, port uint16) *PeerAddress {
	na := PeerAddress{
		Timestamp:    time.Unix(timestamp.Unix(), 0),
		ServicesFlag: serviceFlag,
		IP:           ip,
		Port:         port,
	}
	return &na
}
func NEWPeerAddress(addr *net.TCPAddr, serviceFlag uint64) *PeerAddress {

	return InitPeerAddressIPPort(serviceFlag, addr.IP, addr.Port)
}
func ReadPeerAddress(r io.Reader, pver uint32, na *PeerAddress, ts bool) error {
	var ip [16]byte
	if ts && pver >= protocol.PEER_ADDRESS_TIME_VERSION {
		err := utils.ReadElement(r, (protocol.Uint32Time)(&na.Timestamp))
		if err != nil {
			return err
		}
	}
	err := utils.ReadElements(r, &na.ServicesFlag, &ip)
	if err != nil {
		return err
	}
	var port uint16
	err = utils.ReadElement(r, &port)
	if err != nil {
		return err
	}
	*na = PeerAddress{
		Timestamp:    na.Timestamp,
		ServicesFlag: na.ServicesFlag,
		IP:           net.IP(ip[:]),
		Port:         port,
	}
	return nil

}
func WritePeerAddress(w io.Writer, pver uint32, na *PeerAddress, ts bool) (err error) {
	if ts && pver >= protocol.PEER_ADDRESS_TIME_VERSION {
		err = utils.WriteElement(w, uint32(na.Timestamp.Unix()))
		if err != nil {
			return err
		}
	}
	var ip [16]byte
	if na.IP != nil {
		copy(ip[:], na.IP.To16())
	}
	err = utils.WriteElements(w, na.ServicesFlag, ip)
	if err != nil {
		return
	}
	return binary.Write(w, binary.BigEndian, na.Port)

}

func MaxPeerAddressPayload(version uint32) uint32 {
	len := uint32(26)
	if version >= protocol.PEER_ADDRESS_TIME_VERSION {
		len += 4
	}
	return len
}
