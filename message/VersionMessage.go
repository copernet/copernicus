package message

import (
	"copernicus/protocol"
	"time"
	"copernicus/network"
	"fmt"
	"strings"
)

type VersionMessage struct {
	Message
	ProtocolVersion uint32
	ServiceFlag     protocol.ServiceFlag
	Timestamp       time.Time
	RemoteAddress   *network.NetAddress
	LocalAddress    *network.NetAddress
	Nonce           uint64
	UserAgent       string
	LastBlock       int32
	DisableRelayTx  bool
}

func GetNewVersionMessage(localAddr *network.NetAddress, remoteAddr *network.NetAddress, nonce uint64, lastBlock int32) *VersionMessage {
	versionMessage := VersionMessage{
		ProtocolVersion: protocol.BITCOIN_PROTOCOL_VERSION,
		ServiceFlag:     0,
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		RemoteAddress:   remoteAddr,
		LocalAddress:    localAddr,
		Nonce:           nonce,
		UserAgent:       protocol.LocalUserAgent,
		LastBlock:       lastBlock,
		DisableRelayTx:  false,
	}
	return &versionMessage
}
func (msg *VersionMessage) AddUserAgent(name string, version string, notes ...string) error {
	userAgent := fmt.Sprintf("%s:%s", name, version)
	if len(notes) != 0 {
		userAgent = fmt.Sprintf("%s(%s)", userAgent, strings.Join(notes, ";"))
	}
	err := protocol.ValidateUserAgent(userAgent)
	if err != nil {
		return err
	}
	msg.UserAgent = userAgent
	return nil

}
