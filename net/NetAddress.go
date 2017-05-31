package net

import (
	"time"
	"net"
	"io"
	"btcboost/protocol"
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
func InItNetAddress(addr * net.TCPAddr,serviceFlag uint64)*NetAddress  {

	return InitNetAddressIPPort(serviceFlag,addr.IP,addr.Port)
}
func ReadNetAddress(r io.Reader,pver uint32,na *NetAddress,ts bool)error  {
	var ip [16]byte
	if ts && pver>=protocol.NET_ADDRESS_TIME_VERSION{
		err :=read
	}
}