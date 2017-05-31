package io

import (
	"io"
	"encoding/binary"
	"crypto/rand"
	"time"
	"btcboost/protocol"
	"crypto"
)

type BinaryFreeList chan []byte

var binarySerializer BinaryFreeList = make(chan []byte, IO_MAX_SIZE)

func (l BinaryFreeList) BorrowFront8() (buf []byte) {
	select {
	case buf = <-l:
	default:
		buf = make([]byte, 8)

	}
	return buf[:8]
}
// Return puts the provided byte slice back on the free list.  The buffer MUST
// have been obtained via the Borrow function and therefore have a cap of 8.
func (b BinaryFreeList)Return(buf [] byte) {
	select {
	case b <- buf:
	default:
	}
}

func (b BinaryFreeList)Uint8(r io.Reader) (uint8, error) {
	buf := b.BorrowFront8()[:1]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := buf[0]
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList)Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	buf := b.BorrowFront8()[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint16(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList)Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	buf := b.BorrowFront8()
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint32(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList)Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	buf := b.BorrowFront8()[:8]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint64(buf)
	b.Return(buf)
	return rv, nil

}
func (b BinaryFreeList)PutUint8(w io.Writer, val uint8) error {
	buf := b.BorrowFront8()[:1]
	buf[0] = val
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}

func (b BinaryFreeList)PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := b.BorrowFront8()[:2]
	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList)PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {

	buf := b.BorrowFront8()[:4]
	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList)PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := b.BorrowFront8()[:8]
	byteOrder.PutUint64(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}

func GetUint64FromReder(r io.Reader) (uint64, error) {
	rv, err := binarySerializer.Uint64(r, binary.BigEndian)
	if err != nil {
		return 0, err
	}
	return rv, nil
}
func RandomUint64() (uint64, error) {

	return GetUint64FromReder(rand.Reader)
}

func ReadElement(r io.Reader, element interface{}) (error) {

	switch e := element.(type) {
	case *int32:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return
		}
		*e = int32(rv)
		return nil
	case *uint32:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *int64:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil
	case *uint64:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil
	case *bool:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil
	case *uint32Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = uint32Time(time.Unix(int64(rv), 0))
		return nil
	case *int64Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64Time(time.Unix(int64(rv), 0))
		return nil
	// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return
	case *[protocol.COMMANG_SIZE]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	case *crypto.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	case *protocol.ServiceFlag:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.ServiceFlag(rv)
		return nil
	case protocol.InvType:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.InvType(rv)
		return nil
	case *protocol.BitcoinNet:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = protocol.BitcoinNet(rv)
		return nil


	case *protocol.BloomUpdateType:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = protocol.BloomUpdateType(rv)
		return nil
	case protocol.RejectCode:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = protocol.RejectCode(rv)
		return nil

	}
	return binary.Read(r, binary.LittleEndian, element)

}