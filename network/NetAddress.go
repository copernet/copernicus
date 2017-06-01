package network

import (
	"time"
	"net"
	"io"
	"btcboost/protocol"
	"btcboost/store"
	"encoding/binary"
)

type NetAddress struct {
	Timestamp    time.Time
	ServicesFlag uint64
	IP           net.IP
	Port         int
}

func (na *NetAddress) EqualService(serviceFlag uint64) bool {
	return na.ServicesFlag & serviceFlag == serviceFlag
}

func (na *NetAddress)AddService(serviceFlag uint64) {
	na.ServicesFlag |= serviceFlag
}
func InitNetAddressIPPort(serviceFlag uint64, ip net.IP, port int) *NetAddress {
	return InitNetAddress(time.Now(), serviceFlag, ip, port)
}

func InitNetAddress(timestamp time.Time, serviceFlag uint64, ip net.IP, port int) *NetAddress {
	na := NetAddress{
		Timestamp:time.Unix(timestamp.Unix(), 0),
		ServicesFlag:serviceFlag,
		IP:ip,
		Port:port,
	}
	return &na
}
func InItNetAddress(addr *net.TCPAddr, serviceFlag uint64) *NetAddress {

	return InitNetAddressIPPort(serviceFlag, addr.IP, addr.Port)
}
func ReadNetAddress(r io.Reader, pver uint32, na *NetAddress, ts bool) error {
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
	*na = NetAddress{
		Timestamp:na.Timestamp,
		ServicesFlag:na.ServicesFlag,
		IP:net.IP(ip[:]),
		Port:port,
	}
	return nil

}
func WriteNetAddress(w io.Writer, pver uint32, na *NetAddress, ts bool) (err error) {
	if ts&&pver >= protocol.NET_ADDRESS_TIME_VERSION {
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