package network

import (
	"time"
	"net"
	"io"
	"copernicus/protocol"
	"encoding/binary"
	"copernicus/utils"
	"github.com/btcsuite/go-socks/socks"
	"strconv"
)

type PeerAddress struct {
	Timestamp    time.Time
	ServicesFlag protocol.ServiceFlag
	IP           net.IP
	Port         uint16
}
type PeerAddressFunc func(remoteAddr *PeerAddress) *PeerAddress

func (na *PeerAddress) EqualService(serviceFlag protocol.ServiceFlag) bool {
	return na.ServicesFlag&serviceFlag == serviceFlag
}
func (peerAddrss *PeerAddress) IsIPv4() bool {
	return peerAddrss.IP.To4() != nil
}
func (peerAddress *PeerAddress) IsLocal() bool {

	return peerAddress.IP.IsLoopback() || Zero4Net.Contains(peerAddress.IP)
}

// IsOnionCatTor returns whether or not the passed address is in the IPv6 range
// used by bitcoin to support Tor (fd87:d87e:eb43::/48).  Note that this range
// is the same range used by OnionCat, which is part of the RFC4193 unique local
// IPv6 range.

func (peerAddress *PeerAddress) IsOnionCatTor() bool {
	return OnionCatNet.Contains(peerAddress.IP)

}
func (na *PeerAddress) AddService(serviceFlag protocol.ServiceFlag) {
	na.ServicesFlag |= serviceFlag
}
func InitPeerAddressIPPort(serviceFlag protocol.ServiceFlag, ip net.IP, port uint16) *PeerAddress {
	return InitPeerAddress(time.Now(), serviceFlag, ip, port)
}

func InitPeerAddress(timestamp time.Time, serviceFlag protocol.ServiceFlag, ip net.IP, port uint16) *PeerAddress {
	na := PeerAddress{
		Timestamp:    time.Unix(timestamp.Unix(), 0),
		ServicesFlag: serviceFlag,
		IP:           ip,
		Port:         port,
	}
	return &na
}
func NewPeerAddress(addr *net.TCPAddr, serviceFlag protocol.ServiceFlag) *PeerAddress {

	return InitPeerAddressIPPort(serviceFlag, addr.IP, uint16(addr.Port))
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

func InitPeerAddressWithNetAddr(address net.Addr, servicesFlag protocol.ServiceFlag) (*PeerAddress, error) {
	if tcpAddress, ok := address.(*net.TCPAddr); ok {

		netAddress := NewPeerAddress(tcpAddress, servicesFlag)
		return netAddress, nil

	}
	if proxiedAddr, ok := address.(*socks.ProxiedAddr); ok {
		ip := net.ParseIP(proxiedAddr.Host)
		if ip == nil {

			ip = net.ParseIP("0.0.0.0")
		}
		port := uint16(proxiedAddr.Port)
		peerAddress := InitPeerAddressIPPort(servicesFlag, ip, port)
		return peerAddress, nil
	}
	host, portStr, err := net.SplitHostPort(address.String())
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	peerAddress := InitPeerAddressIPPort(servicesFlag, ip, uint16(port))
	return peerAddress, nil
}
