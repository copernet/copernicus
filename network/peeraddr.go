package network

import (
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/btcboost/copernicus/protocol"
	"github.com/btcsuite/go-socks/socks"
)

type HostToNetAddrFunc func(host string, port uint16, serviceFlag protocol.ServiceFlag) (*PeerAddress, error)

type PeerAddress struct {
	Timestamp    time.Time
	ServicesFlag protocol.ServiceFlag
	IP           net.IP
	Port         uint16
}
type PeerAddressFunc func(remoteAddr *PeerAddress) *PeerAddress

func (peerAddress *PeerAddress) EqualService(serviceFlag protocol.ServiceFlag) bool {
	return peerAddress.ServicesFlag&serviceFlag == serviceFlag
}
func (peerAddress *PeerAddress) IsIPv4() bool {
	return peerAddress.IP.To4() != nil
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
func (peerAddress *PeerAddress) IsRFC2544() bool {
	return rfc2544NeT.Contains(peerAddress.IP)
}

// IsRFC1918 returns whether or not the passed address is part of the IPv4
// private network address space as defined by RFC1918 (10.0.0.0/8,
// 172.16.0.0/12, or 192.168.0.0/16).
func (peerAddress *PeerAddress) IsRFC1918() bool {
	for _, rfc := range rfc1918Nets {
		if rfc.Contains(peerAddress.IP) {
			return true
		}
	}
	return false
}

func (peerAddress *PeerAddress) IsRFC3849() bool {
	return rfc3849Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC3927() bool {
	return rfc3927Net.Contains(peerAddress.IP)
}

func (peerAddress *PeerAddress) IsRFC3964() bool {
	return rfc3964Net.Contains(peerAddress.IP)
}

func (peerAddress *PeerAddress) IsRFC4193() bool {
	return rfc4193Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC4380() bool {
	return rfc4380Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC4843() bool {
	return rfc4380Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC4862() bool {
	return rfc4862Net.Contains(peerAddress.IP)
}

func (peerAddress *PeerAddress) IsRFC5737() bool {
	for _, rfc := range rfc5737Net {
		if rfc.Contains(peerAddress.IP) {
			return true
		}
	}

	return false
}
func (peerAddress *PeerAddress) IsRFC6052() bool {
	return rfc6052Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC6145() bool {
	return rfc6145Net.Contains(peerAddress.IP)
}
func (peerAddress *PeerAddress) IsRFC6598() bool {
	return rfc6598Net.Contains(peerAddress.IP)
}

func (peerAddress *PeerAddress) IsValid() bool {
	return peerAddress.IP != nil &&
		!(peerAddress.IP.IsUnspecified() ||
			peerAddress.IP.Equal(net.IPv4bcast))
}

// IsRoutable returns whether or not the passed address is routable over
// the public internet.  This is true as long as the address is valid and is not
// in any reserved ranges.
func (peerAddress *PeerAddress) IsRoutable() bool {
	return peerAddress.IsValid() && !(peerAddress.IsRFC1918() || peerAddress.IsRFC2544() ||
		peerAddress.IsRFC3927() || peerAddress.IsRFC4862() || peerAddress.IsRFC3849() ||
		peerAddress.IsRFC4843() || peerAddress.IsRFC5737() || peerAddress.IsRFC6598() ||
		peerAddress.IsLocal() || (peerAddress.IsRFC4193() && !peerAddress.IsOnionCatTor()))
}

func (peerAddress *PeerAddress) GroupKey() string {
	if peerAddress.IsLocal() {
		return "local"
	}
	if !peerAddress.IsRoutable() {
		return "unroutable"
	}
	if peerAddress.IsIPv4() {
		return peerAddress.IP.Mask(net.CIDRMask(16, 32)).String()
	}
	if peerAddress.IsRFC6145() || peerAddress.IsRFC6052() {
		ip := peerAddress.IP[12:16]
		return ip.Mask(net.CIDRMask(16, 32)).String()
	}

	if peerAddress.IsRFC3964() {
		ip := peerAddress.IP[2:6]
		return ip.Mask(net.CIDRMask(16, 32)).String()

	}
	if peerAddress.IsRFC4380() {
		// teredo tunnels have the last 4 bytes as the v4 address XOR
		// 0xff.
		ip := net.IP(make([]byte, 4))
		for i, byte := range peerAddress.IP[12:16] {
			ip[i] = byte ^ 0xff
		}
		return ip.Mask(net.CIDRMask(16, 32)).String()
	}
	if peerAddress.IsOnionCatTor() {
		// group is keyed off the first 4 bits of the actual onion key.
		return fmt.Sprintf("tor:%d", peerAddress.IP[6]&((1<<4)-1))
	}

	// OK, so now we know ourselves to be addressManager IPv6 address.
	// bitcoind uses /32 for everything, except for Hurricane Electric's
	// (he.net) IP range, which it uses /36 for.
	bits := 32
	if HeNet.Contains(peerAddress.IP) {
		bits = 36
	}

	return peerAddress.IP.Mask(net.CIDRMask(bits, 128)).String()
}

func (peerAddress *PeerAddress) AddService(serviceFlag protocol.ServiceFlag) {
	peerAddress.ServicesFlag |= serviceFlag
}
func NewPeerAddressIPPort(serviceFlag protocol.ServiceFlag, ip net.IP, port uint16) *PeerAddress {
	return NewPeerAddressTimestamp(time.Now(), serviceFlag, ip, port)
}

func NewPeerAddressTimestamp(timestamp time.Time, serviceFlag protocol.ServiceFlag, ip net.IP, port uint16) *PeerAddress {
	na := PeerAddress{
		Timestamp:    time.Unix(timestamp.Unix(), 0),
		ServicesFlag: serviceFlag,
		IP:           ip,
		Port:         port,
	}
	return &na
}
func NewPeerAddress(addr *net.TCPAddr, serviceFlag protocol.ServiceFlag) *PeerAddress {

	return NewPeerAddressIPPort(serviceFlag, addr.IP, uint16(addr.Port))
}
func ReadPeerAddress(r io.Reader, pver uint32, na *PeerAddress, ts bool) error {
	var ip [16]byte
	if ts && pver >= protocol.PeerAddressTimeVersion {
		err := protocol.ReadElement(r, (protocol.Uint32Time)(na.Timestamp))
		if err != nil {
			return err
		}
	}
	err := protocol.ReadElements(r, &na.ServicesFlag, &ip)
	if err != nil {
		return err
	}
	var port uint16
	err = protocol.ReadElement(r, &port)
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
	if ts && pver >= protocol.PeerAddressTimeVersion {
		err = protocol.WriteElement(w, uint32(na.Timestamp.Unix()))
		if err != nil {
			return err
		}
	}
	var ip [16]byte
	if na.IP != nil {
		copy(ip[:], na.IP.To16())
	}
	err = protocol.WriteElements(w, na.ServicesFlag, ip)
	if err != nil {
		return
	}
	return binary.Write(w, binary.BigEndian, na.Port)

}

func MaxPeerAddressPayload(version uint32) uint32 {
	length := uint32(26)
	if version >= protocol.PeerAddressTimeVersion {
		length += 4
	}
	return length
}

func NewPeerAddressWithNetAddr(address net.Addr, servicesFlag protocol.ServiceFlag) (*PeerAddress, error) {
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
		peerAddress := NewPeerAddressIPPort(servicesFlag, ip, port)
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
	peerAddress := NewPeerAddressIPPort(servicesFlag, ip, uint16(port))
	return peerAddress, nil
}
func (peerAddress *PeerAddress) NetAddressKey() string {
	port := strconv.FormatUint(uint64(peerAddress.Port), 10)
	return net.JoinHostPort(peerAddress.IPString(), port)
}

func (peerAddress *PeerAddress) IPString() string {
	if peerAddress.IsOnionCatTor() {
		base32String := base32.StdEncoding.EncodeToString(peerAddress.IP[6:])
		return strings.ToLower(base32String) + ".onion"
	}
	return peerAddress.IP.String()
}
