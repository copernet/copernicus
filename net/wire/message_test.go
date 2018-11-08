package wire

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/copernet/copernicus/conf"
	"net"
	"os"
	"testing"
	"time"

	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"github.com/davecgh/go-spew/spew"
)

// makeHeader is a convenience function to make a message header in the form of
// a byte slice.  It is used to force errors when reading messages.
func makeHeader(btcnet BitcoinNet, command string,
	payloadLen uint32, checksum uint32) []byte {

	// The length of a bitcoin message header is 24 bytes.
	// 4 byte magic number of the bitcoin network + 12 byte command + 4 byte
	// payload length + 4 byte checksum.
	buf := make([]byte, 24)
	binary.LittleEndian.PutUint32(buf, uint32(btcnet))
	copy(buf[4:], []byte(command))
	binary.LittleEndian.PutUint32(buf[16:], payloadLen)
	binary.LittleEndian.PutUint32(buf[20:], checksum)
	return buf
}

var blockOne block.Block

func getBlockOne() {
	hexStr := "010000006fe28c0ab6f1b372c1a6a246ae63f74f931e8365e15a089c68d6190000000000982051fd1e4ba744bbbe680e1fee14677ba1a3c3540bf7b1cdb606e857233e0e61bc6649ffff001d01e362990101000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0704ffff001d0104ffffffff0100f2052a0100000043410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac00000000"
	bs, _ := hex.DecodeString(hexStr)
	blockOne.Decode(bytes.NewBuffer(bs))
}

func init() {
	getBlockOne()
}

func TestMain(m *testing.M) {
	conf.Cfg = conf.InitConfig([]string{})
	os.Exit(m.Run())
}

// TestMessage tests the Read/WriteMessage and Read/WriteMessageN API.
func TestMessage(t *testing.T) {
	pver := ProtocolVersion

	// Create the various types of messages to test.

	// MsgVersion.
	addrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you := NewNetAddress(addrYou, SFNodeNetwork)
	you.Timestamp = time.Time{} // Version message has zero value timestamp.
	addrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := NewNetAddress(addrMe, SFNodeNetwork)
	me.Timestamp = time.Time{} // Version message has zero value timestamp.
	msgVersion := NewMsgVersion(me, you, 123123, 0)

	msgVerack := NewMsgVerAck()
	msgGetAddr := NewMsgGetAddr()
	msgAddr := NewMsgAddr()
	msgGetBlocks := NewMsgGetBlocks(&util.Hash{})
	msgBlock := (*MsgBlock)(&blockOne)
	msgInv := NewMsgInv()
	msgGetData := NewMsgGetData()
	msgNotFound := NewMsgNotFound()
	//msgTx := NewMsgTx(1)
	msgTx := (*MsgTx)(tx.NewTx(0, 1))
	msgPing := NewMsgPing(123123)
	msgPong := NewMsgPong(123123)
	msgGetHeaders := NewMsgGetHeaders()
	msgHeaders := NewMsgHeaders()
	msgAlert := NewMsgAlert([]byte("payload"), []byte("signature"))
	msgMemPool := NewMsgMemPool()
	msgFilterAdd := NewMsgFilterAdd([]byte{0x01})
	msgFilterClear := NewMsgFilterClear()
	msgFilterLoad := NewMsgFilterLoad([]byte{0x01}, 10, 0, BloomUpdateNone)
	//bh := NewBlockHeader(1, &util.Hash{}, &util.Hash{}, 0, 0)
	bh := block.NewBlockHeader()
	bh.Version = 1
	bh.Time = uint32(time.Now().Unix())
	msgMerkleBlock := NewMsgMerkleBlock(bh)
	msgReject := NewMsgReject("block", errcode.RejectDuplicate, "duplicate block")

	tests := []struct {
		in     Message    // Value to encode
		out    Message    // Expected decoded value
		pver   uint32     // Protocol version for wire encoding
		btcnet BitcoinNet // Network to use for wire encoding
		bytes  int        // Expected num bytes read/written
	}{
		{msgVersion, msgVersion, pver, MainNet, 122},
		{msgVerack, msgVerack, pver, MainNet, 24},
		{msgGetAddr, msgGetAddr, pver, MainNet, 24},
		{msgAddr, msgAddr, pver, MainNet, 25},
		{msgGetBlocks, msgGetBlocks, pver, MainNet, 61},
		{msgBlock, msgBlock, pver, MainNet, 239},
		{msgInv, msgInv, pver, MainNet, 25},
		{msgGetData, msgGetData, pver, MainNet, 25},
		{msgNotFound, msgNotFound, pver, MainNet, 25},
		{msgTx, msgTx, pver, MainNet, 34},
		{msgPing, msgPing, pver, MainNet, 32},
		{msgPong, msgPong, pver, MainNet, 32},
		{msgGetHeaders, msgGetHeaders, pver, MainNet, 61},
		{msgHeaders, msgHeaders, pver, MainNet, 25},
		{msgAlert, msgAlert, pver, MainNet, 42},
		{msgMemPool, msgMemPool, pver, MainNet, 24},
		{msgFilterAdd, msgFilterAdd, pver, MainNet, 26},
		{msgFilterClear, msgFilterClear, pver, MainNet, 24},
		{msgFilterLoad, msgFilterLoad, pver, MainNet, 35},
		{msgMerkleBlock, msgMerkleBlock, pver, MainNet, 110},
		{msgReject, msgReject, pver, MainNet, 79},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		nw, err := WriteMessageN(&buf, test.in, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

		// Ensure the number of bytes written match the expected value.
		if nw != test.bytes {
			t.Errorf("WriteMessage #%d unexpected num bytes "+
				"written - got %d, want %d", i, nw, test.bytes)
		}

		// Decode from wire format.
		rbuf := bytes.NewReader(buf.Bytes())
		nr, msg, _, err := ReadMessageN(rbuf, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}

		// Ensure the number of bytes read match the expected value.
		if nr != test.bytes {
			t.Errorf("ReadMessage #%d unexpected num bytes read - "+
				"got %d, want %d", i, nr, test.bytes)
		}
	}

	// Do the same thing for Read/WriteMessage, but ignore the bytes since
	// they don't return them.
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		err := WriteMessage(&buf, test.in, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("WriteMessage #%d error %v", i, err)
			continue
		}

		// Decode from wire format.
		rbuf := bytes.NewReader(buf.Bytes())
		msg, _, err := ReadMessage(rbuf, test.pver, test.btcnet)
		if err != nil {
			t.Errorf("ReadMessage #%d error %v, msg %v", i, err,
				spew.Sdump(msg))
			continue
		}
	}
}
