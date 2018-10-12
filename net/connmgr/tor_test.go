package connmgr

import (
	"bytes"
	"net"
	"sync"
	"testing"
)

func TestTorLookupIP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:9050")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		addrs, err := TorLookupIP("www.google.com", "127.0.0.1:9050")
		if err != nil {
			t.Fatalf("lookup failed: %v\n", err)
		}
		if addrs[0].String() != "8.8.8.8" {
			t.Fatalf("ip should be 8.8.8.8")
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("accept error: %v\n", err)
	}
	defer conn.Close()
	buf := make([]byte, 10)
	if _, err := conn.Read(buf[0:3]); err != nil {
		t.Fatalf("read error: %v\n", err)
	}
	if !bytes.Equal(buf[0:3], []byte{'\x05', '\x01', '\x00'}) {
		t.Fatalf("handshake failed")
	}
	copy(buf, []byte{'\x05', '\x00'})
	conn.Write(buf[0:2])
	if _, err := conn.Read(buf[0:5]); err != nil {
		t.Fatalf("read error: %v\n", err)
	}
	if !bytes.Equal(buf[0:4], []byte{'\x05', '\xf0', '\x00', '\x03'}) {
		t.Fatalf("invalid req header")
	}
	dlen := int(buf[4])
	domain := make([]byte, dlen)
	if n, err := conn.Read(domain); n != dlen {
		t.Fatalf("read domain name error: %v", err)
	}
	copy(buf, []byte{'\x05', '\x00', '\x00', '\x01'})
	conn.Write(buf[0:4])
	ip := net.ParseIP("8.8.8.8")
	conn.Write(ip[12:16])
	wg.Wait()
}
