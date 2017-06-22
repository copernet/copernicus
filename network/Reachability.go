package network

type Reachability int

const (
	Unreachable Reachability = 0
	Default     Reachability = iota
	Teredo
	Ipv6Weak
	Ipv4
	Ipv6Strong
	Private
)

func GetReachabilityFrom(localPeerAddress, remoteAddress *PeerAddress) Reachability {
	if !remoteAddress.IsRoutable() {
		return Unreachable
	}
	if remoteAddress.IsOnionCatTor() {
		if localPeerAddress.IsOnionCatTor() {
			return Private
		}
		if localPeerAddress.IsRoutable() && localPeerAddress.IsIPv4() {
			return Ipv4
		}
		return Default
	}
	if remoteAddress.IsRFC4380() {
		if !localPeerAddress.IsRoutable() {
			return Default
		}
		if localPeerAddress.IsRFC4380() {
			return Teredo
		}
		if localPeerAddress.IsIPv4() {
			return Ipv4
		}
		return Ipv6Weak

	}
	if remoteAddress.IsIPv4() {
		if localPeerAddress.IsRoutable() && localPeerAddress.IsIPv4() {
			return Ipv4
		}
		return Unreachable
	}
	var tunnelled bool
	if localPeerAddress.IsRFC3964() || localPeerAddress.IsRFC6052() || localPeerAddress.IsRFC6145() {
		tunnelled = true
	}
	if !localPeerAddress.IsRoutable() {
		return Default
	}
	if localPeerAddress.IsRFC4380() {
		return Teredo
	}
	if localPeerAddress.IsIPv4() {
		return Ipv4
	}
	if tunnelled {
		return Ipv6Weak
	}
	return Ipv6Strong

}
