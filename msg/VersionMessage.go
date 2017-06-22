package msg

import (
	"copernicus/protocol"
	"time"
	
	"fmt"
	"strings"
	"io"
	"bytes"
	"copernicus/utils"
	"copernicus/network"
)

type VersionMessage struct {
	Message
	ProtocolVersion uint32
	ServiceFlag     protocol.ServiceFlag
	Timestamp       time.Time
	RemoteAddress   *network.PeerAddress
	LocalAddress    *network.PeerAddress
	Nonce           uint64
	UserAgent       string
	LastBlock       int32
	DisableRelayTx  bool
}

func GetNewVersionMessage(localAddr *network.PeerAddress, remoteAddr *network.PeerAddress, nonce uint64, lastBlock int32) *VersionMessage {
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
func (versionMessage *VersionMessage) HasService(serviceFlag protocol.ServiceFlag) bool {
	return versionMessage.ServiceFlag&serviceFlag == serviceFlag
}

func (versionMessage *VersionMessage) AddService(serviceFlag protocol.ServiceFlag) {
	
	versionMessage.ServiceFlag |= serviceFlag
}
func (versionMessage *VersionMessage) BitcoinParse(reader io.Reader, pver uint32) error {
	buf, ok := reader.(*bytes.Buffer)
	if !ok {
		return fmt.Errorf("version message bitcoin parse reader is not a *bytes.Buffer")
		
	}
	err := utils.ReadElements(buf, versionMessage.ProtocolVersion, versionMessage.ServiceFlag, (*protocol.Int64Time)(versionMessage.Timestamp))
	if err != nil {
		return err
	}
	err = network.ReadPeerAddress(buf, pver, versionMessage.RemoteAddress, false)
	if err != nil {
		return err
	}
	if buf.Len() > 0 {
		err = network.ReadPeerAddress(buf, pver, versionMessage.LocalAddress, false)
		if err != nil {
			return err
		}
	}
	if buf.Len() > 0 {
		err := utils.ReadElement(buf, versionMessage.Nonce)
		if err != nil {
			return err
		}
		
	}
	if buf.Len() > 0 {
		userAgent, err := utils.ReadVarString(buf, pver)
		if err != nil {
			return err
		}
		err = protocol.ValidateUserAgent(userAgent)
		if err != nil {
			return err
		}
		versionMessage.UserAgent = userAgent
	}
	if buf.Len() > 0 {
		err = utils.ReadElement(buf, versionMessage.LastBlock)
		if err != nil {
			return err
		}
	}
	if buf.Len() > 0 {
		var relayTx bool
		utils.ReadElement(reader, &relayTx)
		versionMessage.DisableRelayTx = !relayTx
	}
	return nil
}

func (versionMessage *VersionMessage) BitcoinSerialize(w io.Writer, size uint32) error {
	err := protocol.ValidateUserAgent(versionMessage.UserAgent)
	if err != nil {
		return err
	}
	err = utils.WriteElements(w, versionMessage.ProtocolVersion, versionMessage.ServiceFlag, versionMessage.Timestamp.Unix())
	if err != nil {
		return err
	}
	err = network.WritePeerAddress(w, size, versionMessage.RemoteAddress, false)
	if err != nil {
		return err
	}
	err = network.WritePeerAddress(w, size, versionMessage.LocalAddress, false)
	if err != nil {
		return err
	}
	err = utils.WriteElement(w, versionMessage.Nonce)
	if err != nil {
		return err
	}
	err = utils.WriteVarString(w, size, versionMessage.UserAgent)
	if err != nil {
		return err
	}
	err = utils.WriteElement(w, versionMessage.LastBlock)
	if err != nil {
		return err
	}
	if size >= protocol.BIP0037_VERSION {
		err = utils.WriteElement(w, !versionMessage.DisableRelayTx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (versionMessage *VersionMessage) MaxPayloadLength(pver uint32) uint32 {
	return 33 + (network.MaxPeerAddressPayload(pver) * 2) + MAX_VAR_INT_PAYLOAD + MAX_USERAGENT_LEN
}
func (versionMessage *VersionMessage) Command() string {
	return COMMAND_VERSION
}
