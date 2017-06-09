package msg

import (
	"time"
	"net"
	"io"
	"copernicus/protocol"
	"copernicus/store"
	"encoding/binary"
)

type PeerAddress struct {
	Timestamp    time.Time
	ServicesFlag uint64
	IP           net.IP
	Port         uint16
}
type NetAddressFunc func(remoteAddr *PeerAddress) *PeerAddress

func (na *PeerAddress) EqualService(serviceFlag uint64) bool {
	return na.ServicesFlag&serviceFlag == serviceFlag
}

func (na *PeerAddress) AddService(serviceFlag uint64) {
	na.ServicesFlag |= serviceFlag
}
func InitNetAddressIPPort(serviceFlag uint64, ip net.IP, port uint16) *PeerAddress {
	return InitNetAddress(time.Now(), serviceFlag, ip, port)
}

func InitNetAddress(timestamp time.Time, serviceFlag uint64, ip net.IP, port uint16) *PeerAddress {
	na := PeerAddress{
		Timestamp:    time.Unix(timestamp.Unix(), 0),
		ServicesFlag: serviceFlag,
		IP:           ip,
		Port:         port,
	}
	return &na
}
func InItNetAddress(addr *net.TCPAddr, serviceFlag uint64) *PeerAddress {

	return InitNetAddressIPPort(serviceFlag, addr.IP, addr.Port)
}
func ReadNetAddress(r io.Reader, pver uint32, na *PeerAddress, ts bool) error {
	var ip [16]byte
	if ts && pver >= protocol.NET_ADDRESS_TIME_VERSION {
		err := store.ReadElement(r, (protocol.Uint32Time)(&na.Timestamp))
		if err != nil {
			return err
		}
	}
	err := store.ReadElements(r, &na.ServicesFlag, &ip)
	if err != nil {
		return err
	}
	var port uint16
	err = store.ReadElement(r, &port)
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
func WriteNetAddress(w io.Writer, pver uint32, na *PeerAddress, ts bool) (err error) {
	if ts && pver >= protocol.NET_ADDRESS_TIME_VERSION {
		err = store.WriteElement(w, uint32(na.Timestamp.Unix()))
		if err != nil {
			return err
		}
	}
	var ip [16]byte
	if na.IP != nil {
		copy(ip[:], na.IP.To16())
	}
	err = store.WriteElements(w, na.ServicesFlag, ip)
	if err != nil {
		return
	}
	return binary.Write(w, binary.BigEndian, na.Port)

}
