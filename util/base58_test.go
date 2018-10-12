package util

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

var (
	benchbuf = make([]byte, 1024)
)

func TestMain(m *testing.M) {
	rand.Read(benchbuf)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestBadBase58Decode(t *testing.T) {
	data := "^abcd"
	_, err := bigintBase58Decode(data)
	if err != ErrInvalidCharacter {
		t.Error("expect ErrInvalidCharacter")
	}

	_, err = trezorBase58Decode(data)
	if err != ErrInvalidCharacter {
		t.Error("expect ErrInvalidCharacter")
	}
}

func TestBase58Encode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  []byte
		expect string
	}{
		{
			[]byte{}, "",
		},
		{
			[]byte{32}, "Z",
		},
		{
			[]byte{45}, "n",
		},
		{
			[]byte{48}, "q",
		},
		{
			[]byte{49}, "r",
		},
		{
			[]byte{57}, "z",
		},
		{
			[]byte{45, 49}, "4SU",
		},
		{
			[]byte{49, 49}, "4k8",
		},
	}

	for _, test := range tests {
		rv := bigintBase58Encode(test.input)
		if rv != test.expect {
			t.Errorf("expect %s got %s", test.expect, rv)
		}
	}

	for _, test := range tests {
		rv := trezorBase58Encode(test.input)
		if rv != test.expect {
			t.Errorf("expect %s got %s", test.expect, rv)
		}
	}
}

func TestBase58Decode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expect []byte
		input  string
	}{
		{
			[]byte{}, "",
		},
		{
			[]byte{32}, "Z",
		},
		{
			[]byte{45}, "n",
		},
		{
			[]byte{48}, "q",
		},
		{
			[]byte{49}, "r",
		},
		{
			[]byte{57}, "z",
		},
		{
			[]byte{45, 49}, "4SU",
		},
		{
			[]byte{49, 49}, "4k8",
		},
	}

	for _, test := range tests {
		rv, err := bigintBase58Decode(test.input)
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(rv, test.expect) {
			t.Errorf("expect %x got %x", test.expect, rv)
		}
	}
}

func TestBase58EncodeAndDecode(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 128)
	for i := 0; i < 128; i++ {
		_, err := rand.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		data, _ := bigintBase58Decode(bigintBase58Encode(buf))
		if !bytes.Equal(buf, data) {
			t.Fatalf("expect %x got %x", buf, data)
		}
	}

	zerobuf := make([]byte, 128)
	for i := 0; i < 128; i++ {
		_, err := rand.Read(zerobuf[8:])
		if err != nil {
			t.Fatal(err)
		}

		data, _ := bigintBase58Decode(bigintBase58Encode(zerobuf))
		if !bytes.Equal(zerobuf, data) {
			t.Fatalf("expect %x got %x", zerobuf, data)
		}
	}

	buf = make([]byte, 128)
	for i := 0; i < 128; i++ {
		_, err := rand.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		data, _ := trezorBase58Decode(trezorBase58Encode(buf))
		if !bytes.Equal(buf, data) {
			t.Fatalf("expect %x got %x", buf, data)
		}
	}

	zerobuf = make([]byte, 128)
	for i := 0; i < 128; i++ {
		_, err := rand.Read(zerobuf[8:])
		if err != nil {
			t.Fatal(err)
		}

		data, _ := trezorBase58Decode(trezorBase58Encode(zerobuf))
		if !bytes.Equal(zerobuf, data) {
			t.Fatalf("expect %x got %x", zerobuf, data)
		}
	}

	for i := 0; i < 128; i++ {
		onlyzero := make([]byte, i)
		data, err := bigintBase58Decode(bigintBase58Encode(onlyzero))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(onlyzero, data) {
			t.Fatalf("expect %x got %x", zerobuf, data)
		}
	}

	for i := 0; i < 128; i++ {
		onlyzero := make([]byte, i)
		data, err := bigintBase58Decode(trezorBase58Encode(onlyzero))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(onlyzero, data) {
			t.Fatalf("expect %x got %x", zerobuf, data)
		}
	}
}

func TestBase58EncodeAndDecodeFuzzy(t *testing.T) {
	t.Parallel()

	for i := 0; i < 128; i++ {
		buf := make([]byte, rand.Intn(i+1))
		_, err := rand.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		data, _ := bigintBase58Decode(bigintBase58Encode(buf))
		if !bytes.Equal(buf, data) {
			t.Fatalf("expect %x got %x", buf, data)
		}

		data, _ = trezorBase58Decode(trezorBase58Encode(buf))
		if !bytes.Equal(buf, data) {
			t.Fatalf("expect %x got %x", buf, data)
		}
	}
}

func TestBase58Check(t *testing.T) {
	var checkEncodingStringTests = []struct {
		version byte
		in      string
		out     string
	}{
		{20, "", "3MNQE1X"},
		{20, " ", "B2Kr6dBE"},
		{20, "-", "B3jv1Aft"},
		{20, "0", "B482yuaX"},
		{20, "1", "B4CmeGAC"},
		{20, "-1", "mM7eUf6kB"},
		{20, "11", "mP7BMTDVH"},
		{20, "abc", "4QiVtDjUdeq"},
		{20, "1234598760", "ZmNb8uQn5zvnUohNCEPP"},
		{20, "abcdefghijklmnopqrstuvwxyz", "K2RYDcKfupxwXdWhSAxQPCeiULntKm63UXyx5MvEH2"},
		{20, "00000000000000000000000000000000000000000000000000000000000000", "bi1EWXwJay2udZVxLJozuTb8Meg4W9c6xnmJaRDjg6pri5MBAxb9XwrpQXbtnqEoRV5U2pixnFfwyXC8tRAVC8XxnjK"},
	}

	for x, test := range checkEncodingStringTests {
		// test encoding
		if res := Base58EncodeCheck([]byte(test.in), test.version); res != test.out {
			t.Errorf("CheckEncode test #%d failed: got %s, want: %s", x, res, test.out)
		}

		// test decoding
		res, version, err := Base58DecodeCheck(test.out)
		if err != nil {
			t.Errorf("CheckDecode test #%d failed with err: %v", x, err)
		} else if version != test.version {
			t.Errorf("CheckDecode test #%d failed: got version: %d want: %d", x, version, test.version)
		} else if string(res) != test.in {
			t.Errorf("CheckDecode test #%d failed: got: %s want: %s", x, res, test.in)
		}
	}

	// test the two decoding failure cases
	// case 1: checksum error
	_, _, err := Base58DecodeCheck("3MNQE1Y")
	if err != ErrBadChecksum {
		t.Error("Checkdecode test failed, expected ErrChecksum")
	}

	// case 2: invalid formats (string lengths below 5 mean the version byte and/or the checksum
	// bytes are missing).
	testString := ""
	for length := 0; length < 4; length++ {
		// make a string of length `len`
		_, _, err = Base58DecodeCheck(testString)
		if err != ErrInvalidFormat {
			t.Error("Checkdecode test failed, expected ErrInvalidFormat")
		}
	}

	_, _, err = Base58DecodeCheck("^1234576")
	if err != ErrInvalidCharacter {
		t.Error("Checkdecode test failed, expected ErrInvalidCharacter")
	}

}

func BenchmarkBigintBase58EncodeAndDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bigintBase58Decode(bigintBase58Encode(benchbuf))
	}
}

func BenchmarkTrezorBase58EncodeAndDecode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trezorBase58Decode(trezorBase58Encode(benchbuf))
	}
}
