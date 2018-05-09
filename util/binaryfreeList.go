package util

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

const (
	IoMaxSize     = 1024
	MaxSize       = 0x02000000
	maxBorrowSize = 10
)

var (
	littleEndian = binary.LittleEndian
	bigEndian    = binary.BigEndian
)

type BinaryFreeList chan []byte

var BinarySerializer BinaryFreeList = make(chan []byte, IoMaxSize)

func (b BinaryFreeList) Borrow() (buf []byte) {
	select {
	case buf = <-b:
	default:
		buf = make([]byte, maxBorrowSize)

	}
	return buf[:maxBorrowSize]
}

func (b BinaryFreeList) Return(buf []byte) {
	select {
	case b <- buf:
	default:
	}
}

func (b BinaryFreeList) Uint8(r io.Reader) (uint8, error) {
	buf := b.Borrow()[:1]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := buf[0]
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	buf := b.Borrow()[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint16(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	buf := b.Borrow()[:4]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint32(buf)
	b.Return(buf)
	return rv, nil
}
func (b BinaryFreeList) Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	buf := b.Borrow()[:8]
	if _, err := io.ReadFull(r, buf); err != nil {
		b.Return(buf)
		return 0, err
	}
	rv := byteOrder.Uint64(buf)
	b.Return(buf)
	return rv, nil

}
func (b BinaryFreeList) PutUint8(w io.Writer, val uint8) error {
	buf := b.Borrow()[:1]
	buf[0] = val
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}

func (b BinaryFreeList) PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := b.Borrow()[:2]
	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList) PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {

	buf := b.Borrow()[:4]
	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)
	b.Return(buf)
	return err
}
func (b BinaryFreeList) PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := b.Borrow()[:8]
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

func readElement(r io.Reader, element interface{}) error {
	switch e := element.(type) {
	case *int8:
		rv, err := BinarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = int8(rv)
		return nil

	case *uint8:
		rv, err := BinarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int16:
		rv, err := BinarySerializer.Uint16(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int16(rv)
		return nil

	case *uint16:
		rv, err := BinarySerializer.Uint16(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int32:
		rv, err := BinarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil

	case *uint32:
		rv, err := BinarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int64:
		rv, err := BinarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil

	case *uint64:
		rv, err := BinarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *bool:
		rv, err := BinarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil

	// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case *[12]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case *[16]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil
	}
	return binary.Read(r, littleEndian, element)
}

func ReadElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := readElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeElement(w io.Writer, element interface{}) error {
	switch e := element.(type) {
	case int8:
		err := BinarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil

	case uint8:
		err := BinarySerializer.PutUint8(w, e)
		if err != nil {
			return err
		}
		return nil

	case int16:
		err := BinarySerializer.PutUint16(w, littleEndian, uint16(e))
		if err != nil {
			return err
		}
		return nil

	case uint16:
		err := BinarySerializer.PutUint16(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int32:
		err := BinarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case uint32:
		err := BinarySerializer.PutUint32(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := BinarySerializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case uint64:
		err := BinarySerializer.PutUint64(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case bool:
		var err error
		if e {
			err = BinarySerializer.PutUint8(w, 0x01)
		} else {
			err = BinarySerializer.PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil

	// Message header checksum.
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case [12]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	}
	return binary.Write(w, littleEndian, element)
}

func WriteElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}
