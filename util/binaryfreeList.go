package util

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

const (
	IoMaxSize = 1024
	MaxSize   = 0x02000000
)

type BinaryFreeList chan []byte

var BinarySerializer BinaryFreeList = make(chan []byte, IoMaxSize)

func (b BinaryFreeList) BorrowFront8() (buf []byte) {
	select {
	case buf = <-b:
	default:
		buf = make([]byte, 8)

	}
	return buf[:8]
}

// Return puts the provided byte slice back on the free list.  The buffer MUST
// have been obtained via the Borrow function and therefore have a cap of 8.
func (b BinaryFreeList) Return(buf []byte) {
	select {
	case b <- buf:
	default:
	}
}

func (b BinaryFreeList) Uint8(r io.Reader) (uint8, error) {
	buf := b.BorrowFront8()[:1]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := buf[0]
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	buf := b.BorrowFront8()[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint16(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	buf := b.BorrowFront8()[:4]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint32(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	buf := b.BorrowFront8()[:8]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint64(buf)
	b.Return(buf)
	return rv, nil

}
func (b BinaryFreeList) PutUint8(w io.Writer, val uint8) error {
	buf := b.BorrowFront8()[:1]
	buf[0] = val
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}

func (b BinaryFreeList) PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := b.BorrowFront8()[:2]
	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList) PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {

	buf := b.BorrowFront8()[:4]
	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList) PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := b.BorrowFront8()[:8]
	byteOrder.PutUint64(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}

func GetUint64FromReder(r io.Reader) (uint64, error) {
	rv, err := BinarySerializer.Uint64(r, binary.BigEndian)
	if err != nil {
		return 0, err
	}
	return rv, nil
}
func RandomUint64() (uint64, error) {
	return GetUint64FromReder(rand.Reader)
}
